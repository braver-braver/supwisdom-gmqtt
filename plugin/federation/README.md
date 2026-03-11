# Federation

Federation is a kind of clustering mechanism which provides high-availability and horizontal scaling.
In Federation mode, multiple gmqtt brokers can be grouped together and "act as one".
However, it is impossible to fulfill all requirements in MQTT specification in a distributed environment.
There are some limitations:
1. Persistent session cannot be resumed from another node.
2. Clients with same client id can connect to different nodes at the same time and will not be kicked out.

This is because session information only stores in local node and does not share between nodes.

## Features

- **High Availability**: Multiple broker nodes provide redundancy
- **Horizontal Scaling**: Add more nodes to handle increased load
- **Automatic Discovery**: Nodes automatically discover and connect to each other
- **Shared Subscriptions**: Load balancing across federation nodes
- **Session Persistence**: Each node maintains its own session state
- **Gossip Protocol**: Uses Serf for efficient membership management
- **gRPC Communication**: Fast and reliable inter-node messaging

## Quick Start
The following commands will start a two nodes federation, the configuration files can be found [here](./examples).  
Start node1 in Terminal1:
```bash
$ gmqttd start -c path/to/retry_join/node1_config.yml
```
Start node2 in Terminate2:
```bash
$ gmqttd start -c path/to/retry_join/node2_config2.yml
```
After node1 and node2 is started, they will join into one federation atomically. 

We can test the federation with `mosquitto_pub/sub`:
Connect to node2 and subscribe topicA:
```bash
$ mosquitto_sub -t topicA -h <node2-host> -p <node2-port>
```
Connect to node1 and send a message to topicA:
```bash
$ mosquitto_pub -t topicA -m 123 -h <node1-host> -p <node1-port>
```
The `mosquitto_sub` will receive "123" and print it in the terminal.
```bash
$ mosquitto_sub -t topicA -h <node2-host> -p <node2-port>
123
```

## Join Nodes via REST API
Federation provides gRPC/REST API to join/leave and query members information, see [swagger](./swagger/federation.swagger.json) for details.
In addition to join nodes upon starting up, you can join a node into federation by using `Join` API.  

Start node3 with the configuration with empty `retry_join` which means that the node will not join any nodes upon starting up.
```bash
$ gmqttd start -c path/to/retry_join/join_node3_config.yml
```
We can send `Join` request to any nodes in the federation to get node3 joined, for example, sends `Join` request to node1:
```bash
# Replace with your actual node addresses
$ curl -X POST -d '{"hosts":["<node3-gossip-addr>"]}'  'localhost:8083/v1/federation/join'
{}
```
And check the members in federation:
```bash
curl http://localhost:8083/v1/federation/members
{
    "members": [
        {
            "name": "node1",
            "addr": "<node1-gossip-addr>",
            "tags": {
                "fed_addr": "<node1-fed-addr>"
            },
            "status": "STATUS_ALIVE"
        },
        {
            "name": "node2",
            "addr": "<node2-gossip-addr>",
            "tags": {
                "fed_addr": "<node2-fed-addr>"
            },
            "status": "STATUS_ALIVE"
        },
        {
            "name": "node3",
            "addr": "<node3-gossip-addr>",
            "tags": {
                "fed_addr": "<node3-fed-addr>"
            },
            "status": "STATUS_ALIVE"
        }
    ]
}
```
You will see there are 3 nodes ara alive in the federation.

## Configuration
```go
// Config is the configuration for the federation plugin.
type Config struct {
	// NodeName is the unique identifier for the node in the federation. Defaults to hostname.
	NodeName string `yaml:"node_name"`
	// FedAddr is the gRPC server listening address for the federation internal communication.
	// Defaults to :8901.
	// If the port is missing, the default federation port (8901) will be used.
	FedAddr string `yaml:"fed_addr"`
	// AdvertiseFedAddr is used to change the federation gRPC server address that we advertise to other nodes in the cluster.
	// Defaults to "FedAddr" or the private IP address of the node if the IP in "FedAddr" is 0.0.0.0.
	// However, in some cases, there may be a routable address that cannot be bound.
	// If the port is missing, the default federation port (8901) will be used.
	AdvertiseFedAddr string `yaml:"advertise_fed_addr"`
	// GossipAddr is the address that the gossip will listen on, It is used for both UDP and TCP gossip. Defaults to :8902
	GossipAddr string `yaml:"gossip_addr"`
	// AdvertiseGossipAddr is used to change the gossip server address that we advertise to other nodes in the cluster.
	// Defaults to "GossipAddr" or the private IP address of the node if the IP in "GossipAddr" is 0.0.0.0.
	// If the port is missing, the default gossip port (8902) will be used.
	AdvertiseGossipAddr string `yaml:"advertise_gossip_addr"`
	// RetryJoin is the address of other nodes to join upon starting up.
	// If port is missing, the default gossip port (8902) will be used.
	RetryJoin []string `yaml:"retry_join"`
	// RetryInterval is the time to wait between join attempts. Defaults to 5s.
	RetryInterval time.Duration `yaml:"retry_interval"`
	// RetryTimeout is the timeout to wait before joining all nodes in RetryJoin successfully.
	// If timeout expires, the server will exit with error. Defaults to 1m.
	RetryTimeout time.Duration `yaml:"retry_timeout"`
	// SnapshotPath will be pass to "SnapshotPath" in serf configuration.
	// When Serf is started with a snapshot,
	// it will attempt to join all the previously known nodes until one
	// succeeds and will also avoid replaying old user events.
	SnapshotPath string `yaml:"snapshot_path"`
	// RejoinAfterLeave will be pass to "RejoinAfterLeave" in serf configuration.
	// It controls our interaction with the snapshot file.
	// When set to false (default), a leave causes a Serf to not rejoin
	// the cluster until an explicit join is received. If this is set to
	// true, we ignore the leave, and rejoin the cluster on start.
	RejoinAfterLeave bool `yaml:"rejoin_after_leave"`
}
```

## Implementation Details

### Inner-node Communication
Nodes in the same federation communicate with each other through a couple of gRPC streaming apis:
```proto
message Event {
    uint64 id = 1;
    oneof Event {
        Subscribe Subscribe = 2;
        Message message = 3;
        Unsubscribe unsubscribe = 4;
    }
}
service Federation {
    rpc Hello(ClientHello) returns (ServerHello){}
    rpc EventStream (stream Event) returns (stream Ack){}
}
```
In general, a node is both Client and Server which implements the `Federation` gRPC service. 
* As Client, the node will send subscribe, unsubscribe and message published events to other nodes if necessary.  
Each event has a EventID, which is incremental and unique in a session. 
* As Server, when receives a event from Client, the node returns an acknowledgement after the event has been handled successfully.

### Session State
The event is designed to be idempotent and will be delivered at least once, just like the QoS 1 message in MQTT protocol.
In order to implement QoS 1 protocol flows, the Client and Server need to associate state with a SessionID, 
this is referred to as the Session State. The Server also stores the federation tree and retained messages as part of the Session State.

The Session State in the Client consists of:
 * Events which have been sent to the Server, but have not been acknowledged.
 * Events pending transmission to the Server.

The Session State in the Server consists of:
 * The existence of a Session, even if the rest of the Session State is empty.
 * The EventID of the next event that the Server is willing to accept.
 * Events which have been received from the Client, but have not sent acknowledged yet.
 
The Session State stores in memory only. When the Client starts, it generates a random UUID as SessionID.
When the Client detects a new node is joined or reconnects to the Server, it sends the `Hello` request which contains the SessionID to perform a handshake.
During the handshake, the Server will check whether the session for the SessionID exists.  

* If the session not exists, the Server sends response with `clean_start=true`. 
* If the session exists, the Server sends response with `clean_start=false` and sets the next EventID that it is willing to accept to `next_event_id`.  

After handshake succeed, the Client will start `EventStream`: 
* If the Client receives `clean_start=true`, it sends all local subscriptions and retained messages to the Server in order to sync the full state.
* If the Client receives `clean_start=false`, it sends events of which the EventID is greater than or equal to `next_event_id`.

### Subscription Tree
Each node in the federation will have two subscription trees, the local tree and the federation tree.
The local tree stores subscriptions for local clients which is managed by gmqtt core and the federation tree stores the subscriptions for remote nodes which is managed by the federation plugin.
The federation tree takes node name as subscriber identifier for subscriptions. 
* When receives a sub/unsub packet from a local client, the node will update it's local tree first and then broadcasts the event to other nodes. 
* When receives sub/unsub event from a remote node, the node will only update it's federation tree.  

With the federation tree, the node can determine which node the incoming message should be routed to. 
For example, Node1 and Node2 are in the same federation. Client1 connects to Node1 and subscribes to topic a/b, the subscription trees of these two nodes are as follows:  

Node1 local tree:  

| subscriber | topic | 
|------------|-------|
| client1 | a/b |

Node1 federation tree:    
empty.

Node2 local tree:  
empty.

Node2 federation tree:  
    
| subscriber | topic | 
|------------|-------|
| node1 | a/b |

### Message Distribution Process
When an MQTT client publishes a message, the node where it is located queries the federation tree 
and forwards the message to the relevant node according to the message topic, 
and then the relevant node retrieves the local subscription tree and sends the message to the relevant subscriber.

### Membership Management
Federation uses [Serf](https://github.com/hashicorp/serf) to manage membership.

#### Node States
- **Alive**: Node is healthy and participating in the federation
- **Failed**: Node is unreachable (temporary network issue)
- **Left**: Node has gracefully left the federation

#### Failure Detection
Serf uses a gossip-based failure detection mechanism:
- Periodic health checks between nodes
- Configurable timeout and retry parameters
- Automatic removal of failed nodes after timeout

## Troubleshooting

### Common Issues

#### Port Conflicts
**Problem**: `Failed to start TCP listener: bind: address already in use`

**Solution**: Ensure `fed_addr` and `gossip_addr` ports are not in use:
```bash
# Check if ports are in use
netstat -tuln | grep <port>

# Change ports in configuration
federation:
  fed_addr: "<host>:<port>"
  gossip_addr: "<host>:<port>"
```

#### Nodes Not Joining
**Problem**: Nodes fail to join the federation

**Solution**:
1. Check network connectivity between nodes
2. Verify `retry_join` addresses are correct
3. Ensure firewall allows traffic on gossip ports
4. Check logs for detailed error messages

#### Split Brain
**Problem**: Federation splits into multiple isolated groups

**Solution**:
- Ensure stable network connectivity
- Use proper `retry_join` configuration
- Monitor Serf logs for partition events
- Consider using `snapshot_path` for persistence

### Logging

Federation plugin uses structured logging with the following fields:
- `module`: Always "federation"
- `op`: Operation name (e.g., "event_stream", "retry_join", "reconnect_stream")
- `error`: Error message if operation failed
- `error_chain`: Full error chain for debugging

Example log entries:
```
level=error module=federation op=retry_join error="retry timeout: connection refused" error_chain=["retry timeout: connection refused", "connection refused"]
level=warn module=federation op=reconnect_stream remote_node=node2 reconnect_count=3 error="stream broken"
level=info module=federation op=member_joined node_name=node3
```

## Performance Considerations

### Network Bandwidth
- Each published message may be forwarded to multiple nodes
- Shared subscriptions reduce redundant message forwarding
- Consider network topology when deploying federation

### Latency
- Inter-node communication adds latency to message delivery
- Use low-latency network connections between nodes
- Deploy nodes in the same data center when possible

### Scalability
- Federation scales horizontally by adding more nodes
- Each node handles its own client connections
- Subscription tree size grows with number of topics and nodes

## Best Practices

1. **Use Stable Node Names**: Set explicit `node_name` instead of relying on hostname
2. **Configure Retry Parameters**: Adjust `retry_interval` and `retry_timeout` based on network conditions
3. **Enable Snapshots**: Use `snapshot_path` to persist membership information
4. **Monitor Health**: Use Prometheus metrics to monitor federation health
5. **Plan Network Topology**: Deploy nodes with reliable network connectivity
6. **Use Shared Subscriptions**: Leverage shared subscriptions for load balancing
7. **Test Failure Scenarios**: Regularly test node failures and network partitions

## API Reference

See [swagger documentation](./swagger/federation.swagger.json) for complete API reference.

### Key Endpoints

- `POST /v1/federation/join` - Join nodes to federation
- `POST /v1/federation/leave` - Leave federation gracefully
- `POST /v1/federation/force-leave` - Force remove a failed node
- `GET /v1/federation/members` - List all federation members

## Examples

See [examples directory](./examples) for complete configuration examples:
- `node1_config.yml` - First node with retry_join
- `node2_config.yml` - Second node joining via retry_join
- `join_node3_config.yml` - Node joining via API

## Development

### Running Tests

```bash
# Run all federation tests
go test -v ./plugin/federation/

# Run specific test
go test -v ./plugin/federation/ -run TestFederation_OnMsgArrivedWrapper

# Run with race detection
go test -race ./plugin/federation/
```

### Generating Mocks

```bash
# Generate all mocks
go generate ./plugin/federation/

# Or use the mock generation script
./mock_gen.sh
```

### Protocol Buffers

The federation protocol is defined in `protos/federation.proto`. To regenerate:

```bash
# Install protoc and plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest

# Generate code
protoc --go_out=. --go-grpc_out=. --grpc-gateway_out=. protos/federation.proto
```

## References

- [Serf Documentation](https://www.serf.io/docs/)
- [gRPC Documentation](https://grpc.io/docs/)
- [MQTT Shared Subscriptions](https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901250)

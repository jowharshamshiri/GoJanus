# GoJanus Channel Removal TODO

## PRIME DIRECTIVE: Channels are COMPLETELY REMOVED from the protocol
- No backward compatibility
- No optional support
- Complete removal required

## Files with Channel References

### 1. pkg/manifest/manifest.go
- [ ] Line 14: Add missing `Channels map[string]*ChannelManifest` field to Manifest struct
- [ ] Lines 20-26: Remove `ChannelManifest` struct entirely
- [ ] Lines 97-113: Remove `HasRequest` method (uses channelID)
- [ ] Lines 117-129: Remove/refactor `GetRequest` method (uses channelID)
- [ ] Line 519: Remove channel count validation (already commented out)

### 2. pkg/models/socket_protocol.go
- [ ] Line 15: Remove `ChannelID` field from `JanusRequest` struct
- [ ] Line 43: Update `NewJanusRequest` function signature to remove channelID parameter
- [ ] Line 49: Remove channelID assignment in NewJanusRequest
- [ ] Lines 120-165: Remove/refactor `RequestHandle` struct and methods that reference channel
- [ ] Line 127: Update `NewRequestHandle` to remove channel parameter
- [ ] Lines 142-145: Remove `GetChannel` method

### 3. pkg/protocol/janus_client.go
- [ ] Line 22: Remove `channelID` field from `JanusClient` struct
- [ ] Line 63: Update `validateConstructorInputs` to remove channelID parameter
- [ ] Line 177: Update `New` function signature to remove channelID parameter
- [ ] Line 766: Remove `CreateChannelProxy` method entirely
- [ ] Update all references to `client.channelID` throughout the file

### 4. pkg/core/security_validator.go
- [ ] Line 93: Remove `ValidateChannelID` method
- [ ] Line 334: Remove `ValidateReservedChannelName` method
- [ ] Line 478: Remove `ValidateReservedChannels` method

### 5. pkg/manifest/response_validator.go
- [ ] Line 54: Update `ValidateRequestResponse` signature to remove channelID
- [ ] Line 485: Update `CreateMissingManifestError` signature to remove channelID

### 6. pkg/protocol/message_framing.go
- [ ] Check for any channel validation in message framing

### 7. pkg/server/janus_server.go
- [ ] Check for channel handling in server implementation

### 8. pkg/manifest/manifest_parser.go
- [ ] Check for channel parsing logic

### 9. pkg/core/janus_client.go
- [ ] Check for channel usage in core client

## Test Files Requiring Updates

### Tests in tests/ directory (12 files identified):
- timeout_test.go
- stateless_communication_test.go
- server_features_test.go
- security_test.go
- request_handler_test.go
- message_framing_test.go
- manifest_parser_test.go
- janus_test.go
- janus_client_test.go
- core_socket_communication_test.go
- automatic_id_management_test.go
- advanced_client_features_test.go

## Implementation Order

1. **Update Data Structures** (Priority 1)
   - Remove ChannelID from JanusRequest
   - Remove ChannelManifest struct
   - Update Manifest struct

2. **Update Core Functions** (Priority 2)
   - Update NewJanusRequest signature
   - Update JanusClient.New signature
   - Remove channel validation methods

3. **Update Protocol Layer** (Priority 3)
   - Remove channelID from client struct
   - Update all method signatures
   - Remove ChannelProxy

4. **Update Tests** (Priority 4)
   - Update all test cases to not use channels
   - Remove channel-specific tests

## Verification Steps

1. Run `go build ./...` to ensure compilation
2. Run `go test ./...` to ensure tests pass
3. Run cross-platform tests to verify compatibility
4. Check that manifest responses don't include channels
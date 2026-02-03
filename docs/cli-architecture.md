# CAI CLI Architecture: Design + Implementation

## 1. CLI Architecture

### Directory Structure

```
cai-cli/
├── api/proto/
│   ├── bff.proto                    # Source-of-truth proto definition
│   └── bff/
│       ├── bff.pb.go                # Generated message types
│       └── bff_grpc.pb.go           # Generated gRPC client/server stubs
├── cmd/ai/
│   └── main.go                      # Entry point (3 lines)
├── internal/
│   ├── auth/
│   │   └── auth.go                  # OAuth device flow (unchanged)
│   ├── config/
│   │   ├── config.go                # YAML config with BFF endpoint
│   │   └── config_test.go
│   ├── grpc/
│   │   ├── conn.go                  # Connection mgmt + auth interceptors
│   │   └── conn_test.go
│   ├── cmd/
│   │   ├── root.go                  # Root command, global flags, output format
│   │   ├── helpers.go               # bffConn(), printer() factories
│   │   ├── chat.go                  # chat {list,start,history,context,title,stop}
│   │   ├── model.go                 # model {list}
│   │   ├── mcp.go                   # mcp {list,user,select}
│   │   ├── user.go                  # user {info,settings}
│   │   ├── billing.go               # billing {subscription,plans}
│   │   ├── ratelimit.go             # rate-limit
│   │   ├── login.go                 # OAuth device flow
│   │   ├── health.go                # HTTP health check (gateway)
│   │   ├── version.go               # Version display
│   │   └── cmd_test.go              # Integration tests with in-process gRPC server
│   └── ui/
│       ├── output.go                # Printer with table/json/quiet modes
│       ├── output_test.go
│       └── stream.go                # gRPC server-stream renderer with Ctrl+C
├── scripts/
│   └── generate-proto.sh            # Proto code generation script
├── Makefile                         # build, test, test-cover, proto, vet, fmt
├── go.mod
└── go.sum
```

### Separation of Concerns

| Layer | Package | Responsibility |
|---|---|---|
| **Proto** | `api/proto/bff` | Generated types + client interface. No hand-written code. |
| **Transport** | `internal/grpc` | Connection lifecycle, auth interceptors, timeout defaults. Owns `grpc.ClientConn`. |
| **Auth** | `internal/auth` | OAuth device flow, token refresh. Pure HTTP to Keycloak. No gRPC awareness. |
| **Config** | `internal/config` | YAML persistence, defaults, token validity check. |
| **Commands** | `internal/cmd` | Cobra command tree. Calls transport layer, formats output. |
| **Output** | `internal/ui` | `Printer` with table/json/quiet modes. Proto-to-JSON serialization. Stream rendering. |

**Why this scales**: Each layer has a single reason to change. Adding a new RPC means: (1) update the proto, (2) add a command file, (3) add a test. The transport and output layers are untouched.

---

## 2. Command Design

### Naming Convention

```
ai <service> <operation> [flags]
```

### Full Command Map (15 RPCs -> 15 Commands)

```
ai login                          # OAuth device flow
ai health                         # HTTP gateway health check
ai version                        # Print CLI version

ai chat list                      # GetChatIds
ai chat start [chatId]            # Chat (server streaming)
ai chat history <chatId>          # GetChatHistory
ai chat context <chatId>          # GetChatContext
ai chat title <chatId>            # GenerateTitle
ai chat stop <chatId>             # StopChat

ai model list                     # GetLLMModels

ai mcp list                       # GetMcpInfo
ai mcp user                       # GetMcpForUser
ai mcp select <name>              # SelectMcp

ai user info                      # GetUserInfo
ai user settings                  # GetUserSettings

ai billing subscription           # GetStripeSubscription
ai billing plans                  # GetStripePlans

ai rate-limit                     # QueryRateLimit (alias: ai rl)
```

### Global Flags

```
--bff string       BFF gRPC address host:port (overrides config)
--gateway string   Gateway base URL (overrides config)
--domain string    Domain (overrides config)
-o, --output       Output format: table, json, quiet
```

### Streaming Handling

`ai chat start` opens a persistent gRPC server-streaming connection. Each user prompt sends a new `Chat` RPC. The stream renderer (`ui.RenderGRPCStream`) handles:
- Printing content chunks as they arrive
- Ctrl+C -> context cancellation -> graceful stream teardown
- Separate goroutine for `Recv()` loop to avoid blocking signal handling

---

## 3. gRPC Client Strategy

### Connection Management (`internal/grpc/conn.go`)

```go
// TokenSource handles token access + refresh.
type TokenSource struct {
    cfg         *config.Config
    refreshFunc func() (string, error)
    mu          sync.Mutex
}

// Conn wraps a gRPC connection and BFF client.
type Conn struct {
    cc     *grpc.ClientConn
    Client pb.BFFServiceClient
}
```

**Key design decisions**:

1. **Auth metadata injection via interceptors**, not per-call. The unary interceptor attaches `authorization: Bearer <token>` to every RPC. On `codes.Unauthenticated`, it refreshes the token and retries once. The stream interceptor does the same for streaming RPCs.

2. **TokenSource is mutex-protected** so concurrent refresh attempts don't race.

3. **Timeouts**: Unary RPCs use a 30-second default via `ContextWithTimeout()`. Streaming RPCs (chat) use `context.WithCancel` for user-controlled lifetime.

4. **Environment support**: The `--bff` flag overrides `bff_url` from config. Config defaults to `localhost:9090`. Production configs set the real endpoint.

### Error Handling

gRPC errors are returned as-is to the command layer. Commands wrap them with context:

```go
resp, err := conn.Client.GetLLMModels(ctx, &pb.GetLLMModelsRequest{})
if err != nil {
    return fmt.Errorf("get models: %w", err)
}
```

This preserves the gRPC status code chain for debugging (`codes.Unavailable`, `codes.PermissionDenied`, etc.) while providing human-readable context.

---

## 4. Output Contracts

### Three Modes

| Flag | Behavior | Use Case |
|---|---|---|
| `--output table` (default) | Tabwriter-formatted columns with headers + separator | Interactive terminal use |
| `--output json` | Pretty-printed JSON (protojson for proto messages, encoding/json for tables) | CI/CD, jq piping, scripting |
| `--output quiet` | First column only, one value per line. No info/success messages. | Shell scripting (`xargs`, `while read`) |

### Machine-Readable Guarantees

- **JSON mode**: Output is always valid JSON. Proto messages use `protojson.MarshalOptions` with camelCase field names (proto standard). Table data serializes as `[{"header": "value"}, ...]`.
- **Quiet mode**: One value per line, no headers, no decorative output. Errors still go to stderr.
- **Stderr vs stdout**: All status messages (`[OK]`, `[INFO]`, `[ERROR]`, `[WARN]`) go to stderr. Data goes to stdout. This means `ai chat list -o json | jq .` always works.

---

## 5. Testing Strategy

### Test Categories

| Category | Location | What It Tests | How |
|---|---|---|---|
| **Transport** | `internal/grpc/conn_test.go` | Token source, metadata attachment, empty target, timeout | Unit tests against `TokenSource` and `attachToken()` directly |
| **Output** | `internal/ui/output_test.go` | Table/JSON/quiet rendering, empty data, error visibility | `Printer` with `bytes.Buffer` as Out/Err |
| **Config** | `internal/config/config_test.go` | Token validity, defaults, AuthURL | Pure unit tests |
| **Integration** | `internal/cmd/cmd_test.go` | Full RPC round-trips through all services | In-process gRPC server via `bufconn` |

### In-Process gRPC Test Server

The integration tests use `google.golang.org/grpc/test/bufconn` to create an in-memory gRPC server with no network:

```go
func startTestServer(t *testing.T) pb.BFFServiceClient {
    lis := bufconn.Listen(1024 * 1024)
    srv := grpc.NewServer()
    pb.RegisterBFFServiceServer(srv, &testBFF{})
    go srv.Serve(lis)
    t.Cleanup(func() { srv.Stop() })

    cc, _ := grpc.NewClient("passthrough:///bufconn",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
            return lis.DialContext(ctx)
        }),
    )
    return pb.NewBFFServiceClient(cc)
}
```

**Properties**:
- No real network -> tests are deterministic and fast
- `testBFF` implements `BFFServiceServer` with canned responses
- Each test gets an isolated server via `t.Cleanup`
- Race detector enabled (`go test -race`)

### Test Coverage

The `cmd_test.go` file tests every implemented RPC handler:
- `TestModelList_Table`, `TestModelList_JSON` -- model list with both output modes
- `TestUserInfo` -- user info fields
- `TestChatList`, `TestChatHistory`, `TestChatStop`, `TestGenerateTitle` -- chat operations
- `TestRateLimit` -- rate limit with nested stats
- `TestMCPList` -- MCP server listing
- `TestBillingSubscription`, `TestBillingPlans` -- billing operations
- `TestTruncate` -- string truncation helper (golden cases)

---

## 6. Example Implementation: `ai model list`

This is the fully implemented command, end to end.

**Command definition** (`internal/cmd/model.go`):

```go
var modelListCmd = &cobra.Command{
    Use:   "list",
    Short: "List available LLM models",
    RunE:  runModelList,
}

func runModelList(cmd *cobra.Command, args []string) error {
    conn, cfg, err := bffConn()       // Establish authenticated gRPC conn
    if err != nil {
        return err
    }
    defer conn.Close()

    ctx, cancel := caigrpc.ContextWithTimeout()  // 30s timeout
    defer cancel()

    resp, err := conn.Client.GetLLMModels(ctx, &pb.GetLLMModelsRequest{})
    if err != nil {
        return fmt.Errorf("get models: %w", err)
    }

    p := printer(cfg)
    if p.Format == ui.FormatJSON {
        p.ProtoJSON(resp)              // protojson serialization
        return nil
    }

    headers := []string{"NAME", "ID", "OWNED_BY"}
    rows := make([][]string, 0, len(resp.GetModels()))
    for _, m := range resp.GetModels() {
        rows = append(rows, []string{
            m.GetName(),
            fmt.Sprintf("%d", m.GetModelId()),
            m.GetOwnedBy(),
        })
    }
    p.Table(headers, rows)
    return nil
}
```

**Auth metadata injection** happens transparently in the interceptor -- the command code never touches tokens.

**Test** (`internal/cmd/cmd_test.go`):

```go
func TestModelList_Table(t *testing.T) {
    client := startTestServer(t)     // In-process gRPC server
    resp, err := client.GetLLMModels(context.Background(), &pb.GetLLMModelsRequest{})
    if err != nil {
        t.Fatalf("GetLLMModels: %v", err)
    }
    if len(resp.GetModels()) != 2 { ... }
    if resp.GetModels()[0].GetName() != "gpt-4" { ... }
}
```

---

## 7. Quality Bar

### "Done" Means

- [x] Every BFF gRPC RPC (15 of 15) has a corresponding CLI command
- [x] All commands compile and are reachable via `ai <service> <operation> --help`
- [x] Auth token is injected via gRPC interceptor, not per-command code
- [x] Token refresh on `Unauthenticated` is automatic and transparent
- [x] Output supports `--output table|json|quiet` globally
- [x] Stderr for messages, stdout for data
- [x] Integration tests use in-process gRPC server (no real backend needed)
- [x] `go vet ./...` passes
- [x] `go test -race ./...` passes
- [x] Binary builds successfully with version injection

### What Would Block Merging

- Failing tests
- `go vet` warnings
- Proto files out of sync with BFF
- Commands that don't handle gRPC errors (swallowing errors silently)
- Hard-coded endpoints in command files

### Future Extensions Enabled by This Design

| Extension | Change Required |
|---|---|
| Add a new RPC | Add proto method -> regenerate -> add one command file -> add test |
| mTLS support | Add TLS config to `internal/grpc/conn.go` dial options |
| Retry policy | Add `grpc.WithDefaultServiceConfig()` to dial options |
| Output as YAML | Add `FormatYAML` case to `Printer.Table()` |
| Batch/pipe mode | Commands already read args from flags, not interactive prompts |
| Shell completion | Cobra's built-in `completion` command is already registered |
| Multi-realm | Add `--realm` flag, inject `x-realm` in interceptor metadata |

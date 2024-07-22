package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "math/rand"
    "net/http"
    "os"
    "time"
)

type rpcReq struct {
    JSONRPC string        `json:"jsonrpc"`
    ID      int           `json:"id"`
    Method  string        `json:"method"`
    Params  []interface{} `json:"params"`
}

type rpcResp struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      int             `json:"id"`
    Result  json.RawMessage `json:"result"`
    Error   *struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
    } `json:"error,omitempty"`
}

func call(url, method string, params []interface{}, timeout time.Duration) (*rpcResp, error) {
    body, _ := json.Marshal(rpcReq{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
    client := &http.Client{Timeout: timeout}
    resp, err := client.Post(url, "application/json", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    data, _ := io.ReadAll(resp.Body)
    var out rpcResp
    if err := json.Unmarshal(data, &out); err != nil {
        return nil, err
    }
    if out.Error != nil {
        return &out, fmt.Errorf("rpc error: %s", out.Error.Message)
    }
    return &out, nil
}

func main() {
    // env: RPCS=http://rpc1,http://rpc2  RAWTX=0xf86...  ATTEMPTS=3
    rpcs := os.Getenv("RPCS")
    raw := os.Getenv("RAWTX")
    attempts := 3
    if v := os.Getenv("ATTEMPTS"); v != "" {
        fmt.Sscanf(v, "%d", &attempts)
    }
    if rpcs == "" || raw == "" {
        fmt.Println("usage: RPCS=url1,url2 RAWTX=0x.. [ATTEMPTS=3] go run relaynet_gateway.go")
        os.Exit(1)
    }
    urls := []string{}
    for _, u := range bytes.Split([]byte(rpcs), []byte(",")) {
        uu := string(bytes.TrimSpace(u))
        if uu != "" { urls = append(urls, uu) }
    }
    rand.Seed(time.Now().UnixNano())

    var lastErr error
    for i := 0; i < attempts; i++ {
        // shuffle for simple load-spread
        idx := rand.Intn(len(urls))
        url := urls[idx]
        fmt.Printf("attempt %d/%d -> %s\n", i+1, attempts, url)
        // eth_sendRawTransaction
        _, err := call(url, "eth_sendRawTransaction", []interface{}{raw}, 8*time.Second)
        if err == nil {
            fmt.Println("✓ relayed successfully via", url)
            return
        }
        fmt.Println("x failed:", err)
        lastErr = err
        time.Sleep(time.Duration(1+i) * time.Second)
    }
    fmt.Println("❌ all relays failed:", lastErr)
}

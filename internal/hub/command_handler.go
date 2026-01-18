package hub

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/henrygd/beszel/internal/common"
    "github.com/henrygd/beszel/internal/hub/systems"
    "github.com/henrygd/beszel/internal/hub/ws"
    "github.com/pocketbase/pocketbase/apis"
    "github.com/pocketbase/pocketbase/core"

    "github.com/fxamacker/cbor/v2"
)

// CommandRequest represents a command execution request (mirrors agent.CommandRequest)
type CommandRequest struct {
    Command string            `json:"command"`
    Args    []string          `json:"args,omitempty"`
    Timeout int               `json:"timeout"`
    Workdir string            `json:"workdir,omitempty"`
    Env     map[string]string `json:"env,omitempty"`
}

// CommandResponse represents the result of a command execution (mirrors agent.CommandResponse)
type CommandResponse struct {
    ExitCode int    `json:"exitCode"`
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
    Error    string `json:"error,omitempty"`
    Duration int64  `json:"duration"`
}

// CommandHistoryEntry stores information about a executed command (mirrors agent.CommandHistoryEntry)
type CommandHistoryEntry struct {
    Command   string        `json:"command"`
    Args      []string      `json:"args"`
    Timestamp string        `json:"timestamp"`
    Duration  int64         `json:"duration"`
    ExitCode  int           `json:"exitCode"`
    Workdir   string        `json:"workdir"`
}

// RegisterCommandRoutes registers command execution routes
func (h *Hub) registerCommandRoutes(se *core.ServeEvent) error {
    apiAuth := se.Router.Group("/api/sonar")
    apiAuth.Bind(apis.RequireAuth())

    // Execute command on an agent
    apiAuth.POST("/agents/:id/commands/execute", h.executeCommand)
    // Get command history from an agent
    apiAuth.GET("/agents/:id/commands/history", h.getCommandHistory)

    return nil
}

// executeCommand handles POST /api/sonar/agents/:id/commands/execute
func (h *Hub) executeCommand(e *core.RequestEvent) error {
    // Get agent ID from path
    agentID := e.Request.PathValue("id")
    if agentID == "" {
        return e.JSON(http.StatusBadRequest, map[string]string{"error": "agent ID is required"})
    }

    // Parse request body
    var req CommandRequest
    body, readErr := io.ReadAll(e.Request.Body)
    if readErr != nil {
        return e.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("failed to read request body: %s", readErr.Error())})
    }
    if err := json.Unmarshal(body, &req); err != nil {
        return e.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid request body: %s", err.Error())})
    }

    // Validate command
    if req.Command == "" {
        return e.JSON(http.StatusBadRequest, map[string]string{"error": "command is required"})
    }

    // Set default timeout
    if req.Timeout <= 0 {
        req.Timeout = 30
    }
    if req.Timeout > 300 {
        req.Timeout = 300 // max 5 minutes
    }

    // Get system from system manager
    system, err := h.sm.GetSystem(agentID)
    if err != nil {
        return e.JSON(http.StatusNotFound, map[string]string{"error": "agent not found"})
    }

    // Check if agent is online
    if system.Status != "online" {
        return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "agent is offline"})
    }

    // Execute command via WebSocket or SSH
    response, err := h.executeCommandOnAgent(system, req)
    if err != nil {
        // Check if it's a timeout or agent offline error
        if errors.Is(err, context.DeadlineExceeded) {
            return e.JSON(http.StatusGatewayTimeout, map[string]string{"error": "command execution timed out"})
        }
        return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }

    return e.JSON(http.StatusOK, response)
}

// getCommandHistory handles GET /api/sonar/agents/:id/commands/history
func (h *Hub) getCommandHistory(e *core.RequestEvent) error {
    // Get agent ID from path
    agentID := e.Request.PathValue("id")
    if agentID == "" {
        return e.JSON(http.StatusBadRequest, map[string]string{"error": "agent ID is required"})
    }

    // Parse query parameters
    query := e.Request.URL.Query()
    limitStr := query.Get("limit")
    var limit int
    if limitStr != "" {
        _, err := fmt.Sscanf(limitStr, "%d", &limit)
        if err != nil || limit <= 0 || limit > 100 {
            limit = 10 // default
        }
    } else {
        limit = 10 // default
    }

    // Get system from system manager
    system, err := h.sm.GetSystem(agentID)
    if err != nil {
        return e.JSON(http.StatusNotFound, map[string]string{"error": "agent not found"})
    }

    // Check if agent is online
    if system.Status != "online" {
        return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "agent is offline"})
    }

    // Get command history from agent
    history, err := h.getCommandHistoryFromAgent(system, limit)
    if err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }

    return e.JSON(http.StatusOK, map[string]any{"history": history})
}

// executeCommandOnAgent executes a command on a specific agent
func (h *Hub) executeCommandOnAgent(sys *systems.System, req CommandRequest) (*CommandResponse, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Prefer WebSocket if available
    if sys.WsConn != nil && sys.WsConn.IsConnected() {
        return h.executeCommandViaWebSocket(ctx, sys.WsConn, req)
    }

    return nil, errors.New("no connection available to agent")
}

// executeCommandViaWebSocket executes a command via WebSocket connection
func (h *Hub) executeCommandViaWebSocket(ctx context.Context, wsConn *ws.WsConn, req CommandRequest) (*CommandResponse, error) {
    // Marshal request to CBOR
    reqBytes, err := cbor.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal command request: %w", err)
    }

    // Wait for response
    var response CommandResponse
    handler := &commandResponseHandler{result: &response}
    err = wsConn.SendAndWait(ctx, common.ExecuteCommand, reqBytes, handler)
    if err != nil {
        return nil, err
    }

    return &response, nil
}

// commandResponseHandler handles command execution responses
type commandResponseHandler struct {
    result *CommandResponse
}

func (h *commandResponseHandler) Handle(agentResponse common.AgentResponse) error {
    if agentResponse.Error != "" {
        h.result.Error = agentResponse.Error
    }
    return cbor.Unmarshal(agentResponse.Data, h.result)
}

func (h *commandResponseHandler) HandleLegacy(rawData []byte) error {
    return cbor.Unmarshal(rawData, h.result)
}

// executeCommandViaSSH executes a command via SSH connection
func (h *Hub) executeCommandViaSSH(ctx context.Context, sys *systems.System, req CommandRequest) (*CommandResponse, error) {
    // This is a placeholder for SSH-based command execution
    // In a full implementation, this would use SSH transport to execute commands
    return nil, errors.New("SSH command execution not yet implemented")
}

// getCommandHistoryFromAgent retrieves command history from an agent
func (h *Hub) getCommandHistoryFromAgent(sys *systems.System, limit int) ([]CommandHistoryEntry, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Prefer WebSocket if available
    if sys.WsConn != nil && sys.WsConn.IsConnected() {
        return h.getCommandHistoryViaWebSocket(ctx, sys.WsConn, limit)
    }

    return nil, errors.New("no connection available to agent")
}

// getCommandHistoryViaWebSocket retrieves command history via WebSocket
func (h *Hub) getCommandHistoryViaWebSocket(ctx context.Context, wsConn *ws.WsConn, limit int) ([]CommandHistoryEntry, error) {
    // Marshal request to CBOR
    reqBytes, err := cbor.Marshal(map[string]any{"limit": limit})
    if err != nil {
        return nil, fmt.Errorf("failed to marshal history request: %w", err)
    }

    // Wait for response
    var history []CommandHistoryEntry
    handler := &historyResponseHandler{result: &history}
    err = wsConn.SendAndWait(ctx, common.ExecuteCommand, reqBytes, handler)
    if err != nil {
        return nil, err
    }

    return history, nil
}

// historyResponseHandler handles command history responses
type historyResponseHandler struct {
    result *[]CommandHistoryEntry
}

func (h *historyResponseHandler) Handle(agentResponse common.AgentResponse) error {
    if agentResponse.Error != "" {
        return errors.New(agentResponse.Error)
    }
    return cbor.Unmarshal(agentResponse.Data, h.result)
}

func (h *historyResponseHandler) HandleLegacy(rawData []byte) error {
    return cbor.Unmarshal(rawData, h.result)
}

// getCommandHistoryViaSSH retrieves command history via SSH
func (h *Hub) getCommandHistoryViaSSH(ctx context.Context, sys *systems.System, limit int) ([]CommandHistoryEntry, error) {
    // This is a placeholder for SSH-based history retrieval
    return nil, errors.New("SSH history retrieval not yet implemented")
}

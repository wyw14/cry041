package httptransport

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/wyw14/cry041/internal/application"
	"github.com/wyw14/cry041/internal/domain"
)

type Handler struct {
	commands *application.ReleaseService
	queries  *application.QueryService
	validate *validator.Validate
}

func NewRouter(commands *application.ReleaseService, queries *application.QueryService, ready func() error, middlewares ...gin.HandlerFunc) *gin.Engine {
	h := &Handler{commands: commands, queries: queries, validate: validator.New()}
	r := gin.New()
	r.Use(middlewares...)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		if err := ready(); err != nil {
			c.JSON(503, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})
	v1 := r.Group("/api/v1")
	v1.POST("/releases", h.create)
	v1.POST("/releases/:id/submit", h.submit)
	v1.POST("/releases/:id/signoffs", h.sign)
	v1.POST("/releases/:id/evaluate", h.evaluate)
	v1.POST("/releases/:id/publish", h.publish)
	v1.POST("/releases/:id/execute", h.execute)
	v1.GET("/dashboard", h.dashboard)
	v1.GET("/releases/:id/timeline", h.timeline)
	return r
}
func actor(c *gin.Context) string {
	v := c.GetHeader("X-Actor-ID")
	if v == "" {
		return "demo-owner"
	}
	return v
}
func actorRole(c *gin.Context) string { return c.GetHeader("X-Actor-Role") }
func requireRole(c *gin.Context, allowed ...string) bool {
	role := actorRole(c)
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	fail(c, http.StatusForbidden, "FORBIDDEN", errors.New("当前角色无权执行该操作"))
	return false
}
func revision(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.GetHeader("If-Match"), 10, 64)
}
func (h *Handler) create(c *gin.Context) {
	var in application.CreateRelease
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "INVALID_JSON", err)
		return
	}
	in.IdempotencyKey = c.GetHeader("Idempotency-Key")
	if err := h.validate.Struct(in); err != nil {
		fail(c, 422, "VALIDATION_FAILED", err)
		return
	}
	out, err := h.commands.Create(c.Request.Context(), in, actor(c))
	respond(c, out, err)
}
func (h *Handler) submit(c *gin.Context) {
	rev, err := revision(c)
	if err != nil {
		fail(c, 422, "VERSION_REQUIRED", err)
		return
	}
	out, err := h.commands.Submit(c.Request.Context(), c.Param("id"), rev, actor(c))
	respond(c, out, err)
}
func (h *Handler) sign(c *gin.Context) {
	if !requireRole(c, "quality", "operations", "security") {
		return
	}
	rev, err := revision(c)
	if err != nil {
		fail(c, 422, "VERSION_REQUIRED", err)
		return
	}
	var in domain.Signoff
	if err = c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "INVALID_JSON", err)
		return
	}
	in.ActorID = actor(c)
	in.Role = actorRole(c)
	out, err := h.commands.Sign(c.Request.Context(), c.Param("id"), rev, in)
	respond(c, out, err)
}
func (h *Handler) evaluate(c *gin.Context) {
	if !requireRole(c, "release_manager") {
		return
	}
	rev, err := revision(c)
	if err != nil {
		fail(c, 422, "VERSION_REQUIRED", err)
		return
	}
	var in struct {
		RequiredRoles []string `json:"required_roles" validate:"required,min=1"`
	}
	if err = c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "INVALID_JSON", err)
		return
	}
	out, err := h.commands.Evaluate(c.Request.Context(), c.Param("id"), rev, in.RequiredRoles, actor(c))
	respond(c, out, err)
}
func (h *Handler) publish(c *gin.Context) {
	if !requireRole(c, "release_manager") {
		return
	}
	rev, err := revision(c)
	if err != nil {
		fail(c, 422, "VERSION_REQUIRED", err)
		return
	}
	out, err := h.commands.Publish(c.Request.Context(), c.Param("id"), rev, actor(c))
	respond(c, out, err)
}
func (h *Handler) execute(c *gin.Context) {
	if !requireRole(c, "operations") {
		return
	}
	out, err := h.commands.Execute(c.Request.Context(), c.Param("id"), actor(c))
	respond(c, out, err)
}
func (h *Handler) dashboard(c *gin.Context) {
	out, err := h.queries.Dashboard(c.Request.Context(), actor(c))
	respond(c, out, err)
}
func (h *Handler) timeline(c *gin.Context) {
	out, err := h.queries.Timeline(c.Request.Context(), c.Param("id"))
	respond(c, out, err)
}
func respond(c *gin.Context, v any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, v)
		return
	}
	code, status := "BUSINESS_ERROR", 409
	if errors.Is(err, domain.ErrConflict) {
		code = "VERSION_CONFLICT"
	} else if errors.Is(err, domain.ErrSnapshotReadOnly) {
		code = "SNAPSHOT_READ_ONLY"
	} else if errors.Is(err, domain.ErrInvalidTransition) {
		code = "INVALID_TRANSITION"
	}
	fail(c, status, code, err)
}
func fail(c *gin.Context, status int, code string, err error) {
	c.JSON(status, gin.H{"code": code, "message": err.Error(), "field_errors": []any{}, "request_id": c.GetString("request_id")})
}

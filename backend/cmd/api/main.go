package main

import (
	"context"
	"log"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/cors"

	appconfig "scorehub/internal/config"
	"scorehub/internal/http/handlers"
	"scorehub/internal/http/middleware"
	"scorehub/internal/realtime"
	"scorehub/internal/store"
)

func main() {
	cfg := appconfig.Load()

	ctx := context.Background()
	st, err := store.New(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer st.Close()

	hub := realtime.NewHub()
	startAutoEndInactiveScorebooksJob(ctx, st, hub)

	h := server.Default(server.WithHostPorts(cfg.Addr))
	h.Use(middleware.RequestLog())
	h.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PATCH", "OPTIONS"},
		AllowHeaders:    []string{"Authorization", "Content-Type", "X-Dev-OpenID"},
	}))

	authHandlers := handlers.NewAuthHandlers(cfg, st)
	meHandlers := handlers.NewMeHandlers(st)
	scorebookHandlers := handlers.NewScorebookHandlers(cfg, st, hub)
	ledgerHandlers := handlers.NewLedgerHandlers(st)
	locationHandlers := handlers.NewLocationHandlers(cfg)

	api := h.Group("/api/v1")
	auth := api.Group("/auth")
	auth.POST("/dev_login", authHandlers.DevLogin)
	auth.POST("/wechat_login", authHandlers.WechatLogin)

	authed := api.Group("", middleware.AuthRequired(cfg, st))
	authed.GET("/me", meHandlers.GetMe)
	authed.PATCH("/me", meHandlers.UpdateMe)
	authed.POST("/scorebooks", scorebookHandlers.CreateScorebook)
	authed.GET("/scorebooks", scorebookHandlers.ListMyScorebooks)
	authed.GET("/scorebooks/:id", scorebookHandlers.GetScorebookDetail)
	authed.PATCH("/scorebooks/:id", scorebookHandlers.UpdateScorebook)
	authed.POST("/scorebooks/:id/end", scorebookHandlers.EndScorebook)
	authed.POST("/scorebooks/:id/join", scorebookHandlers.JoinScorebook)
	authed.PATCH("/scorebooks/:id/members/me", scorebookHandlers.UpdateMyProfile)
	authed.GET("/scorebooks/:id/invite_qrcode", scorebookHandlers.GetInviteQRCode)
	authed.POST("/scorebooks/:id/records", scorebookHandlers.CreateRecord)
	authed.GET("/scorebooks/:id/records", scorebookHandlers.ListRecords)
	authed.POST("/invites/:code/join", scorebookHandlers.JoinByInviteCode)
	authed.POST("/ledgers", ledgerHandlers.CreateLedger)
	authed.GET("/ledgers", ledgerHandlers.ListLedgers)
	authed.POST("/ledgers/:id/members", ledgerHandlers.AddLedgerMember)
	authed.PATCH("/ledgers/:id/members/:memberId", ledgerHandlers.UpdateLedgerMember)
	authed.POST("/ledgers/:id/records", ledgerHandlers.AddLedgerRecord)
	authed.POST("/ledgers/:id/end", ledgerHandlers.EndLedger)

	// Public: allow location & invite info lookup without login.
	api.GET("/location/reverse_geocode", locationHandlers.ReverseGeocode)
	api.GET("/invites/:code", scorebookHandlers.GetInviteInfo)
	api.GET("/ledgers/:id", ledgerHandlers.GetLedgerDetail)

	h.GET("/ws/scorebooks/:id", scorebookHandlers.ScorebookWS)

	log.Printf("scorehub api listening on %s", cfg.Addr)
	h.Spin()
}

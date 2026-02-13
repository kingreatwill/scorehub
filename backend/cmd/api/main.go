package main

import (
	"context"
	"io/fs"
	"log"
	"mime"
	"path"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/cors"

	"scorehub/assets"
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
	h.GET("/static/*filepath", staticAssetsHandler())

	authHandlers := handlers.NewAuthHandlers(cfg, st)
	meHandlers := handlers.NewMeHandlers(st)
	scorebookHandlers := handlers.NewScorebookHandlers(cfg, st, hub)
	ledgerHandlers := handlers.NewLedgerHandlers(cfg, st)
	birthdayHandlers := handlers.NewBirthdayHandlers(st)
	depositHandlers := handlers.NewDepositHandlers(st)
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
	authed.DELETE("/scorebooks/:id", scorebookHandlers.DeleteScorebook)
	authed.POST("/scorebooks/:id/end", scorebookHandlers.EndScorebook)
	authed.POST("/scorebooks/:id/join", scorebookHandlers.JoinScorebook)
	authed.PATCH("/scorebooks/:id/members/me", scorebookHandlers.UpdateMyProfile)
	authed.GET("/scorebooks/:id/invite_qrcode", scorebookHandlers.GetInviteQRCode)
	authed.POST("/scorebooks/:id/records", scorebookHandlers.CreateRecord)
	authed.GET("/scorebooks/:id/records", scorebookHandlers.ListRecords)
	authed.POST("/invites/:code/join", scorebookHandlers.JoinByInviteCode)
	authed.POST("/ledgers", ledgerHandlers.CreateLedger)
	authed.GET("/ledgers", ledgerHandlers.ListLedgers)
	authed.PATCH("/ledgers/:id", ledgerHandlers.UpdateLedger)
	authed.DELETE("/ledgers/:id", ledgerHandlers.DeleteLedger)
	authed.GET("/ledgers/:id/invite_qrcode", ledgerHandlers.GetInviteQRCode)
	authed.POST("/ledgers/:id/bind", ledgerHandlers.BindLedgerMember)
	authed.POST("/ledgers/:id/members", ledgerHandlers.AddLedgerMember)
	authed.PATCH("/ledgers/:id/members/:memberId", ledgerHandlers.UpdateLedgerMember)
	authed.POST("/ledgers/:id/records", ledgerHandlers.AddLedgerRecord)
	authed.POST("/ledgers/:id/end", ledgerHandlers.EndLedger)
	authed.POST("/birthdays", birthdayHandlers.CreateBirthday)
	authed.GET("/birthdays", birthdayHandlers.ListBirthdays)
	authed.GET("/birthdays/:id", birthdayHandlers.GetBirthday)
	authed.PATCH("/birthdays/:id", birthdayHandlers.UpdateBirthday)
	authed.DELETE("/birthdays/:id", birthdayHandlers.DeleteBirthday)
	authed.POST("/deposits/accounts", depositHandlers.CreateDepositAccount)
	authed.GET("/deposits/accounts", depositHandlers.ListDepositAccounts)
	authed.GET("/deposits/accounts/:id", depositHandlers.GetDepositAccount)
	authed.PATCH("/deposits/accounts/:id", depositHandlers.UpdateDepositAccount)
	authed.DELETE("/deposits/accounts/:id", depositHandlers.DeleteDepositAccount)
	authed.POST("/deposits/accounts/:id/records", depositHandlers.CreateDepositRecord)
	authed.GET("/deposits/accounts/:id/records", depositHandlers.ListDepositAccountRecords)
	authed.GET("/deposits/records", depositHandlers.ListDepositRecords)
	authed.GET("/deposits/records/:id", depositHandlers.GetDepositRecord)
	authed.PATCH("/deposits/records/:id", depositHandlers.UpdateDepositRecord)
	authed.DELETE("/deposits/records/:id", depositHandlers.DeleteDepositRecord)
	authed.GET("/deposits/tags", depositHandlers.ListDepositTags)
	authed.GET("/deposits/stats", depositHandlers.GetDepositStats)

	// Public: allow location & invite info lookup without login.
	api.GET("/location/reverse_geocode", locationHandlers.ReverseGeocode)
	api.GET("/invites/:code", scorebookHandlers.GetInviteInfo)
	api.GET("/ledgers/:id", ledgerHandlers.GetLedgerDetail)

	h.GET("/ws/scorebooks/:id", scorebookHandlers.ScorebookWS)

	log.Printf("scorehub api listening on %s", cfg.Addr)
	h.Spin()
}

func staticAssetsHandler() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		reqPath := strings.TrimPrefix(string(c.Param("filepath")), "/")
		if reqPath == "" {
			c.SetStatusCode(404)
			return
		}
		if strings.Contains(reqPath, "..") {
			c.SetStatusCode(400)
			return
		}

		data, err := fs.ReadFile(assets.FS, reqPath)
		if err != nil {
			c.SetStatusCode(404)
			return
		}

		if contentType := mime.TypeByExtension(path.Ext(reqPath)); contentType != "" {
			c.SetContentType(contentType)
		}
		c.Response.SetBodyRaw(data)
	}
}

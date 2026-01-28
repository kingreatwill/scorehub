package main

import (
	"context"
	"log"
	"time"

	"scorehub/internal/realtime"
	"scorehub/internal/store"
)

const (
	autoEndInactiveFor = 7 * 24 * time.Hour
	autoEndCheckEvery  = 1 * time.Hour
	autoEndRunTimeout  = 15 * time.Second
)

func startAutoEndInactiveScorebooksJob(ctx context.Context, st *store.Store, hub *realtime.Hub) {
	go func() {
		ticker := time.NewTicker(autoEndCheckEvery)
		defer ticker.Stop()

		run := func() {
			runCtx, cancel := context.WithTimeout(ctx, autoEndRunTimeout)
			defer cancel()

			ended, err := st.AutoEndInactiveScorebooks(runCtx, autoEndInactiveFor)
			if err != nil {
				log.Printf("auto end inactive scorebooks: %v", err)
				return
			}

			for _, sb := range ended {
				var champion any
				var runnerUp any
				var third any
				if tops, err := st.GetTopWinners(runCtx, sb.ID); err == nil {
					if len(tops) >= 1 {
						champion = map[string]any{
							"memberId":  tops[0].ID,
							"nickname":  tops[0].Nickname,
							"avatarUrl": tops[0].AvatarURL,
							"score":     tops[0].Score,
						}
					}
					if len(tops) >= 2 {
						runnerUp = map[string]any{
							"memberId":  tops[1].ID,
							"nickname":  tops[1].Nickname,
							"avatarUrl": tops[1].AvatarURL,
							"score":     tops[1].Score,
						}
					}
					if len(tops) >= 3 {
						third = map[string]any{
							"memberId":  tops[2].ID,
							"nickname":  tops[2].Nickname,
							"avatarUrl": tops[2].AvatarURL,
							"score":     tops[2].Score,
						}
					}
				} else {
					log.Printf("auto end scorebook winners failed: scorebook_id=%s err=%v", sb.ID, err)
				}
				winners := map[string]any{
					"champion": champion,
					"runnerUp": runnerUp,
					"third":    third,
				}

				hub.Broadcast(sb.ID, map[string]any{
					"type": "scorebook.ended",
					"data": map[string]any{
						"id":        sb.ID,
						"endedAt":   sb.EndedAt,
						"updatedAt": sb.UpdatedAt,
						"winners":   winners,
						"autoEnded": true,
					},
				})
			}
		}

		// run once on startup
		run()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

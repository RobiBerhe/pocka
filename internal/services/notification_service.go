package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
	"pocka/ent"
	"pocka/ent/user"
	"pocka/internal/core"
)

type notificationService struct {
	db        *ent.Client
	bot       *tgbotapi.BotAPI
	scheduler *cron.Cron
}

func NewNotificationService(db *ent.Client, bot *tgbotapi.BotAPI) core.NotificationService {
	return &notificationService{
		db:        db,
		bot:       bot,
		scheduler: cron.New(),
	}
}

func (s *notificationService) Start() {
	// Run every hour at the top of the hour
	_, err := s.scheduler.AddFunc("0 * * * *", func() {
		s.checkAndNotify(context.Background())
	})
	if err != nil {
		slog.Error("Failed to schedule notification job", "error", err)
		return
	}

	s.scheduler.Start()
	slog.Info("Notification Service started")
}

func (s *notificationService) Stop() {
	s.scheduler.Stop()
	slog.Info("Notification Service stopped")
}

func (s *notificationService) checkAndNotify(ctx context.Context) {
	slog.Info("Running streak reminder job")
	
	users, err := s.db.User.Query().
		Where(
			user.OnboardingCompleted(true),
			user.CurrentStreakGT(0),
		).All(ctx)
	if err != nil {
		slog.Error("Failed to query users for notifications", "error", err)
		return
	}

	for _, u := range users {
		loc, err := time.LoadLocation(u.Timezone)
		if err != nil {
			continue
		}

		nowLocal := time.Now().In(loc)
		
		// Logic: If it's between 8:00 PM and 9:00 PM in their local time
		// and they haven't logged anything today.
		if nowLocal.Hour() == 20 {
			todayDate := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc)
			
			if u.LastLogDate.IsZero() || !u.LastLogDate.Equal(todayDate) {
				s.sendReminder(u)
			}
		}
	}
}

func (s *notificationService) sendReminder(u *ent.User) {
	msgText := fmt.Sprintf("Hey %s! 🚀\n\nYour *%d day streak* is at risk! 🔥\n\nDon't forget to log a quick expense or income today to keep it alive. Every bit of tracking counts!", u.FullName, u.CurrentStreak)
	
	msg := tgbotapi.NewMessage(u.TelegramID, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	
	_, err := s.bot.Send(msg)
	if err != nil {
		slog.Error("Failed to send reminder", "user_id", u.TelegramID, "error", err)
	} else {
		slog.Info("Sent streak reminder", "user_id", u.TelegramID, "streak", u.CurrentStreak)
	}
}

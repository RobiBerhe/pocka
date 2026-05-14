package services

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"sort"

	"github.com/fogleman/gg"
	"pocka/internal/core"
)

type reportService struct{}

func NewReportService() core.ReportService {
	return &reportService{}
}

func (s *reportService) GenerateWeeklyCard(ctx context.Context, stats *core.WeeklyStats) ([]byte, error) {
	const (
		W = 1200
		H = 900
	)

	dc := gg.NewContext(W, H)

	// 1. Solid Dark Background (Safer than gradient for first fix)
	dc.SetRGB255(18, 18, 18) // Deep Black
	dc.Clear()

	// 2. Subtle Gradient Overlay
	grad := gg.NewLinearGradient(0, 0, W, H)
	grad.AddColorStop(0, color.RGBA{0, 0, 0, 0})
	grad.AddColorStop(1, color.RGBA{40, 20, 80, 100}) // Deep Purple with alpha
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, W, H)
	dc.Fill()

	// 2. Draw Card Border (Glassmorphism effect)
	dc.SetRGBA(1, 1, 1, 0.1)
	dc.DrawRoundedRectangle(40, 40, W-80, H-80, 20)
	dc.Fill()

	// 3. Header
	dc.SetRGB(1, 1, 1)
	
	fontPaths := []string{
		"/System/Library/Fonts/Supplemental/Arial.ttf",     // Mac
		"/usr/share/fonts/ttf-dejavu/DejaVuSans.ttf",      // Alpine A
		"/usr/share/fonts/freefont/FreeSans.ttf",          // Alpine FreeFont
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",          // Alpine B
		"/usr/share/fonts/TTF/DejaVuSans.ttf",              // Alpine C
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", // Debian/Ubuntu
		"/System/Library/Fonts/SFNSMono.ttf",             // Mac Fallback
	}
	
	var fontPath string
	for _, path := range fontPaths {
		if _, err := os.Stat(path); err == nil {
			fontPath = path
			slog.Info("Selected font path", "path", path)
			break
		}
	}
	
	if fontPath == "" {
		slog.Error("No valid fonts found in standard paths")
	}
	
	if err := dc.LoadFontFace(fontPath, 54); err != nil {
		slog.Error("Failed to load font", "path", fontPath, "error", err)
	}
	dc.DrawString("Pocka Weekly Wrap", 100, 150)

	dc.SetRGBA(1, 1, 1, 0.6)
	dc.LoadFontFace(fontPath, 28)
	dateRange := fmt.Sprintf("%s - %s", stats.StartDate.Format("Jan 02"), stats.EndDate.Format("Jan 02"))
	dc.DrawString(dateRange, 100, 200)

	// 4. Streak Badge
	dc.SetRGBA(1, 0.5, 0, 0.2) // Orange glow
	dc.DrawRoundedRectangle(W-350, 110, 250, 60, 30)
	dc.Fill()
	dc.SetRGB(1, 0.5, 0)
	dc.LoadFontFace(fontPath, 28)
	dc.DrawStringAnchored(fmt.Sprintf("%d-DAY STREAK", stats.Streak), W-200, 140, 0.5, 0.5)

	// Draw custom flame icon
	drawFlame(dc, W-320, 140, 30)

	// 5. Total Expense
	dc.SetRGBA(1, 1, 1, 0.7)
	dc.LoadFontFace(fontPath, 28)
	dc.DrawString("TOTAL EXPENSE", 100, 350)
	
	dc.SetRGB(1, 1, 1)
	dc.LoadFontFace(fontPath, 96)
	totalText := fmt.Sprintf("%.0f %s", stats.TotalExpense, stats.Currency)
	dc.DrawString(totalText, 100, 460)

	// 6. Category Breakdown
	dc.SetRGBA(1, 1, 1, 0.8)
	dc.LoadFontFace(fontPath, 34)
	dc.DrawString("Spending Categories", 650, 350)

	// Sort categories by amount
	type catVal struct {
		Name   string
		Amount float64
	}
	var cats []catVal
	for name, val := range stats.CategoryTotals {
		cats = append(cats, catVal{name, val})
	}
	sort.Slice(cats, func(i, j int) bool {
		return cats[i].Amount > cats[j].Amount
	})

	y := 420.0
	colors := []color.RGBA{
		{255, 150, 50, 255},  // Orange
		{50, 200, 255, 255},  // Blue
		{200, 100, 255, 255}, // Purple
		{100, 255, 150, 255}, // Green
	}
	for i, cat := range cats {
		if i >= 4 {
			break
		}
		
		percentage := (cat.Amount / stats.TotalExpense) * 100
		
		// Category Text
		dc.SetRGBA(1, 1, 1, 0.9)
		dc.LoadFontFace(fontPath, 24)
		dc.DrawString(fmt.Sprintf("%.0f%% %s", percentage, cat.Name), 650, y)
		dc.DrawStringAnchored(fmt.Sprintf("%.0f %s", cat.Amount, stats.Currency), W-100, y, 1, 0.5)

		// Progress Bar Background
		dc.SetRGBA(1, 1, 1, 0.1)
		dc.DrawRoundedRectangle(650, y+20, 450, 16, 8)
		dc.Fill()

		// Progress Bar Fill
		dc.SetColor(colors[i%len(colors)])
		dc.DrawRoundedRectangle(650, y+20, 450*(percentage/100), 16, 8)
		dc.Fill()

		y += 90
	}

	// 7. Footer / Branding
	dc.SetRGBA(1, 1, 1, 0.4)
	dc.LoadFontFace(fontPath, 32)
	dc.DrawString("Pocka", 100, H-100)
	dc.DrawStringAnchored("Your Money Assistant", W-100, H-100, 1, 0.5)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func drawFlame(dc *gg.Context, x, y, size float64) {
	dc.Push()
	dc.Translate(x, y)
	
	// Flame shape
	dc.MoveTo(0, size/2)
	dc.CubicTo(-size/2, size/2, -size/2, -size/4, 0, -size/2)
	dc.CubicTo(size/8, -size/4, size/4, -size/8, size/4, 0)
	dc.CubicTo(size/4, size/4, 0, size/2, 0, size/2)
	
	dc.SetRGB(1, 0.5, 0) // Orange
	dc.Fill()
	
	// Inner flame (yellow)
	innerSize := size * 0.6
	dc.MoveTo(0, innerSize/2)
	dc.CubicTo(-innerSize/2, innerSize/2, -innerSize/2, -innerSize/4, 0, -innerSize/2)
	dc.CubicTo(innerSize/8, -innerSize/4, innerSize/4, -innerSize/8, innerSize/4, 0)
	dc.CubicTo(innerSize/4, innerSize/4, 0, innerSize/2, 0, innerSize/2)
	
	dc.SetRGB(1, 0.9, 0) // Yellow
	dc.Fill()
	
	dc.Pop()
}

package consultation_test

import (
	"testing"
	"time"
	_ "time/tzdata" // 実行環境に依存せず、タイムゾーンのデータを使えるようにする

	"github.com/kajiya-i/muda/backend/internal/domain/consultation"
)

func location(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestNewDeadlines(t *testing.T) {
	tokyo := location(t, "Asia/Tokyo")
	newYork := location(t, "America/New_York")

	tests := []struct {
		name         string
		likedAt      time.Time
		loc          *time.Location
		wantPurchase time.Time
		wantReport   time.Time
	}{
		{
			name:         "middle of the month",
			likedAt:      time.Date(2026, 10, 15, 12, 0, 0, 0, tokyo),
			loc:          tokyo,
			wantPurchase: time.Date(2026, 11, 1, 0, 0, 0, 0, tokyo),
			wantReport:   time.Date(2026, 11, 8, 0, 0, 0, 0, tokyo),
		},
		{
			name:         "last moment of the month",
			likedAt:      time.Date(2026, 10, 31, 23, 59, 59, 0, tokyo),
			loc:          tokyo,
			wantPurchase: time.Date(2026, 11, 1, 0, 0, 0, 0, tokyo),
			wantReport:   time.Date(2026, 11, 8, 0, 0, 0, 0, tokyo),
		},
		{
			name:         "december rolls over to the next year",
			likedAt:      time.Date(2026, 12, 20, 9, 0, 0, 0, tokyo),
			loc:          tokyo,
			wantPurchase: time.Date(2027, 1, 1, 0, 0, 0, 0, tokyo),
			wantReport:   time.Date(2027, 1, 8, 0, 0, 0, 0, tokyo),
		},
		{
			name:         "leap year february",
			likedAt:      time.Date(2028, 2, 29, 9, 0, 0, 0, tokyo),
			loc:          tokyo,
			wantPurchase: time.Date(2028, 3, 1, 0, 0, 0, 0, tokyo),
			wantReport:   time.Date(2028, 3, 8, 0, 0, 0, 0, tokyo),
		},
		{
			// UTC では 10 月 31 日だが、東京では 11 月 1 日なので、11 月の期限になる。
			name:         "month is decided by the home time zone",
			likedAt:      time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC),
			loc:          tokyo,
			wantPurchase: time.Date(2026, 12, 1, 0, 0, 0, 0, tokyo),
			wantReport:   time.Date(2026, 12, 8, 0, 0, 0, 0, tokyo),
		},
		{
			// ニューヨークでは 11 月 1 日に夏時間が終わる。期限はその後の現地時刻の 0 時になる。
			name:         "daylight saving time ends within the grace period",
			likedAt:      time.Date(2026, 10, 20, 12, 0, 0, 0, newYork),
			loc:          newYork,
			wantPurchase: time.Date(2026, 11, 1, 0, 0, 0, 0, newYork),
			wantReport:   time.Date(2026, 11, 8, 0, 0, 0, 0, newYork),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := consultation.NewDeadlines(tt.likedAt, tt.loc)
			if !got.Purchase().Equal(tt.wantPurchase) {
				t.Errorf("Purchase = %v, want %v", got.Purchase(), tt.wantPurchase)
			}
			if !got.Report().Equal(tt.wantReport) {
				t.Errorf("Report = %v, want %v", got.Report(), tt.wantReport)
			}
		})
	}
}

func TestReportExpired(t *testing.T) {
	tokyo := location(t, "Asia/Tokyo")
	d := consultation.NewDeadlines(time.Date(2026, 10, 15, 12, 0, 0, 0, tokyo), tokyo)

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "after the purchase deadline, before the report deadline", now: time.Date(2026, 11, 7, 23, 59, 59, 0, tokyo), want: false},
		{name: "exactly at the report deadline", now: time.Date(2026, 11, 8, 0, 0, 0, 0, tokyo), want: true},
		{name: "after the report deadline", now: time.Date(2026, 11, 8, 0, 0, 1, 0, tokyo), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.ReportExpired(tt.now); got != tt.want {
				t.Errorf("ReportExpired(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}

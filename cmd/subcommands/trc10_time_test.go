package cmd

import (
	"bytes"
	"math"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestAssetTime(t *testing.T) {
	valid := int64(1_574_757_000_000) // 2019-11-26T08:30:00Z
	got, ok := assetTime(valid).(time.Time)
	if !ok || !got.Equal(time.Unix(valid/1000, 0)) {
		t.Errorf("assetTime(%d) = %v, want the time.Time", valid, assetTime(valid))
	}

	year10001 := time.Date(10001, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	for _, ms := range []int64{math.MaxInt64, math.MinInt64, year10001} {
		if got := assetTime(ms); got != ms {
			t.Errorf("assetTime(%d) = %v, want the raw milliseconds", ms, got)
		}
		var out bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&out)
		if err := printJSON(cmd, map[string]interface{}{"ICOEnd": assetTime(ms)}); err != nil {
			t.Errorf("printJSON with assetTime(%d): %v", ms, err)
		}
	}
}

// Whatever the local time zone, assetTime must return something printJSON can
// encode, including times that cross the year 0 or 9999 limit only in some
// zones.
func TestAssetTime_YearBoundariesInEveryZone(t *testing.T) {
	boundaries := []time.Time{
		time.Date(9999, 12, 31, 12, 0, 0, 0, time.UTC),
		time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC),
		time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(10000, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(-1, 12, 31, 12, 0, 0, 0, time.UTC),
	}
	for _, zone := range []*time.Location{time.UTC, time.FixedZone("UTC-14", -14*3600), time.FixedZone("UTC+14", 14*3600)} {
		for _, b := range boundaries {
			ms := b.UnixMilli()
			var out bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&out)
			if err := printJSON(cmd, map[string]interface{}{"ICOEnd": assetTimeIn(ms, zone)}); err != nil {
				t.Errorf("zone %s, %s: printJSON(assetTimeIn(%d)) error = %v", zone, b.Format(time.RFC3339), ms, err)
			}
		}
	}
}

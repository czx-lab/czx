package match

import (
	"fmt"
	"testing"
)

func TestMatch(t *testing.T) {
	t.Run("match test", func(t *testing.T) {
		match := NewMatch(nil)
		for i := 1; i < 100; i++ {
			if err := match.Enqueue(fmt.Sprintf("%v", i)); err != nil {
				t.Errorf("err %v", err)
			}
		}

		players, err := match.StartMatching(Matching{
			MatchID: "test",
			Num:     10,
			Timeout: 10,
			Fn: func(playerID string) bool {
				return true
			},
		})
		if err != nil {
			t.Errorf("matching error: %v", err)
		}

		t.Log(players)
	})
}

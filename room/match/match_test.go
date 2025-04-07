package match

import "testing"

func TestMatch(t *testing.T) {
	t.Run("match test", func(t *testing.T) {
		match := NewMatch(nil)
		for i := 1; i < 100; i++ {
			if err := match.Enqueue(uint64(i)); err != nil {
				t.Errorf("err %v", err)
			}
		}

		players, err := match.StartMatching("match_1", 5, nil)
		if err != nil {
			t.Errorf("matching error: %v", err)
		}

		t.Log(players)
	})
}

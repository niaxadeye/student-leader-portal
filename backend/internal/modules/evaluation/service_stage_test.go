package evaluation

import "testing"

func ptrFloat(v float64) *float64 { return &v }
func ptrInt(v int) *int           { return &v }

func TestCombineStandingsRankSum(t *testing.T) {
	t.Parallel()
	rows := combineStandings(CombineRankSum, 1, 1, []combineInput{
		{UserID: "a", FullName: "A", MainRank: ptrInt(1), RemoteRank: ptrInt(2)}, // 3
		{UserID: "b", FullName: "B", MainRank: ptrInt(2), RemoteRank: ptrInt(1)}, // 3
		{UserID: "c", FullName: "C", MainRank: ptrInt(3), RemoteRank: ptrInt(3)}, // 6
		{UserID: "d", FullName: "D", MainRank: ptrInt(1)},                        // no remote
	})
	if rows[0].Rank == nil || *rows[0].Rank != 1 || rows[1].Rank == nil || *rows[1].Rank != 1 {
		t.Fatalf("tied best: %+v %+v", rows[0].Rank, rows[1].Rank)
	}
	if rows[2].Rank == nil || *rows[2].Rank != 3 {
		t.Fatalf("third: %+v", rows[2].Rank)
	}
	if rows[3].Rank != nil || rows[3].Combined != nil {
		t.Fatal("incomplete row must not rank")
	}
}

func TestCombineStandingsScoreSumWeights(t *testing.T) {
	t.Parallel()
	rows := combineStandings(CombineScoreSum, 2, 1, []combineInput{
		{UserID: "a", FullName: "A", MainScore: ptrFloat(10), RemoteScore: ptrFloat(10)}, // 30
		{UserID: "b", FullName: "B", MainScore: ptrFloat(5), RemoteScore: ptrFloat(20)},  // 30
		{UserID: "c", FullName: "C", MainScore: ptrFloat(20), RemoteScore: ptrFloat(1)},  // 41
	})
	if rows[2].Rank == nil || *rows[2].Rank != 1 {
		t.Fatalf("highest weighted score first: %+v", rows[2])
	}
	if rows[0].Rank == nil || *rows[0].Rank != 2 || rows[1].Rank == nil || *rows[1].Rank != 2 {
		t.Fatalf("tie for second: %+v %+v", rows[0].Rank, rows[1].Rank)
	}
}

func TestCombineStandingsRankSumWeights(t *testing.T) {
	t.Parallel()
	rows := combineStandings(CombineRankSum, 2, 1, []combineInput{
		{UserID: "a", MainRank: ptrInt(1), RemoteRank: ptrInt(3)}, // 5
		{UserID: "b", MainRank: ptrInt(2), RemoteRank: ptrInt(1)}, // 5
		{UserID: "c", MainRank: ptrInt(3), RemoteRank: ptrInt(2)}, // 8
	})
	if rows[0].Combined == nil || *rows[0].Combined != 5 || rows[2].Rank == nil || *rows[2].Rank != 3 {
		t.Fatalf("weighted ranks: %+v", rows)
	}
}

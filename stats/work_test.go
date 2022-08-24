package stats

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewWorkStats(t *testing.T) {
	ctx := context.Background()
	s := NewWorkStats("sjdifijwef", rdb)
	err := s.AddPv(ctx, "127.0.0.1", "ua ie edge")
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddUv(ctx, "jdsiofjiwoejfoi")
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddLike(ctx, "jfigiergjierijg")
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddShare(ctx, "jfigiergjierijg")
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddComment(ctx, "asdfjkllafkjsl", "comment1")
	if err != nil {
		t.Fatal(err)
	}

	b, _ := s.InLike(ctx, "jfigiergjierijg")
	if !b {
		t.Fatal("like diff error")
	}
	b, _ = s.InShare(ctx, "jfigiergjierijg")
	if !b {
		t.Fatal("share diff error")
	}
	b, _ = s.InComment(ctx, "asdfjkllafkjsl", "comment1")
	if !b {
		t.Fatal("comment diff error")
	}

	_ = s.UnLike(ctx, "jfigiergjierijg")
	_ = s.AddLike(ctx, "fjdijigfijfgdij")
	_ = s.UnComment(ctx, "asdfjkllafkjsl", "comment1")
	_ = s.AddComment(ctx, "fjdgjierjiergij", "comment2")

	err = s.SummarySync(ctx)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.GetSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := json.Marshal(result)
	t.Log(string(c))

}

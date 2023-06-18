package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fzerorubigd/collection-filter/internal/bgg"
	"github.com/fzerorubigd/collection-filter/internal/cache"
	"github.com/fzerorubigd/collection-filter/internal/mapper"
	"github.com/fzerorubigd/gobgg"
	"go.uber.org/ratelimit"
)

func defPath() string {
	exec, _ := os.Executable()
	path := filepath.Dir(exec)
	return filepath.Join(path, "db.badger")
}

func suggestedPlayerCount(sp []gobgg.SuggestedPlayerCount, pl int) (string, int, float32) {
	for i := range sp {
		if sp[i].NumPlayers == fmt.Sprint(pl) {
			return sp[i].Suggestion()
		}
	}

	return "Not Recommended", 0, 0
}

func main() {

	ctx, cnl := signal.NotifyContext(context.Background(),
		syscall.SIGKILL,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGABRT)
	defer cnl()

	var (
		cacheRoot            string
		userName             string
		cnt                  int
		weightMin, weightMax int
	)

	flag.StringVar(&cacheRoot, "db", defPath(), "the db path to use")
	flag.StringVar(&userName, "username", "fzerorubigd", "user name to get the list for")
	flag.IntVar(&cnt, "player-count", 5, "player count")
	flag.IntVar(&weightMin, "min-weight", 0, "min weight")
	flag.IntVar(&weightMax, "max-weight", 50, "max weight")
	flag.Parse()

	iface, err := cache.NewBadgerCache(cacheRoot)
	if err != nil {
		log.Fatal(err)
	}
	defer iface.Close()
	rl := ratelimit.New(20, ratelimit.Per(60*time.Second))

	api, err := bgg.NewCachedBGGAPI(iface, gobgg.SetLimiter(rl))
	if err != nil {
		log.Fatal(err)
	}

	gm, err := api.GetCollection(ctx, userName, false)
	if err != nil {
		log.Fatal(err)
	}

	da := mapper.Reduce(gm, mapper.RecommendedFor(cnt), mapper.OnlyBoardGame(), mapper.WeightReducer(weightMin, weightMax))
	mapper.Sort(da, mapper.ComparatorPlayerCount(cnt))

	for a := range da {
		fmt.Print(da[a].Name, " => ", "Weight: ", da[a].AverageWeight, " | ", cnt, " Player: ")
		fmt.Println(suggestedPlayerCount(da[a].SuggestedPlayerCount, cnt))
	}

}

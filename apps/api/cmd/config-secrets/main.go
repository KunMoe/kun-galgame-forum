// config-secrets import copies the running API container's environment into
// the config store. It reads `docker inspect` output on stdin and only prints
// its plan unless -apply is given.
//
//	docker inspect <kungal-api container> | docker compose -f docker-compose.prod.yml \
//	  --profile jobs run --rm -T tools config-secrets import [-keys A,B] [-apply]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"kun-galgame-api/internal/envconfig"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"

	"github.com/joho/godotenv"
)

type container struct {
	Name   string
	Config struct {
		Env []string
	}
}

func main() {
	_ = godotenv.Load()
	if len(os.Args) < 2 || os.Args[1] != "import" {
		fmt.Fprintln(os.Stderr, "usage: config-secrets import [-keys NAME,NAME] [-apply] < docker-inspect.json")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	apply := fs.Bool("apply", false, "write the plan; without it only print it")
	keys := fs.String("keys", "", "non-secret variables to store as well, comma separated")
	_ = fs.Parse(os.Args[2:])

	if err := run(os.Stdin, os.Stdout, *keys, *apply); err != nil {
		fmt.Fprintln(os.Stderr, "config-secrets:", err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer, keys string, apply bool) error {
	var cs []container
	if err := json.NewDecoder(in).Decode(&cs); err != nil {
		return fmt.Errorf("read docker inspect from stdin: %w", err)
	}
	if len(cs) != 1 {
		return fmt.Errorf("pipe exactly one container, the kungal-api one; got %d", len(cs))
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if config.StoreKey() == nil {
		return envconfig.ErrStoreOff
	}
	store := envconfig.New(database.NewPostgres(cfg.Database, cfg.Server.Mode), config.StoreKey())
	ctx := context.Background()
	stored, err := store.All(ctx)
	if err != nil {
		return err
	}
	held := map[string]string{}
	for _, s := range stored {
		held[s.Key] = s.Value
	}

	plan := Plan(cs[0].Config.Env, keys)
	note := "imported from " + strings.TrimPrefix(cs[0].Name, "/")
	for _, kv := range plan {
		old, has := held[kv.name]
		verb := "store  "
		switch {
		case has && old == kv.value:
			verb = "same   "
		case has:
			verb = "replace"
		}
		fmt.Fprintf(out, "%s %s = %s\n", verb, kv.name, store.Shown(kv.name, kv.value))
		if apply && verb != "same   " {
			if _, err := store.Set(ctx, kv.name, kv.value, note, envconfig.ActionImport, nil); err != nil {
				return fmt.Errorf("%s: %w", kv.name, err)
			}
		}
	}
	if !apply {
		fmt.Fprintln(out, "dry run: nothing written; run again with -apply")
	}
	return nil
}

type pair struct{ name, value string }

// Plan picks what to store: every secret-looking variable the API reads, plus
// the non-secret ones named in keys. Everything else stays in the compose file.
func Plan(env []string, keys string) []pair {
	extra := map[string]bool{}
	for k := range strings.SplitSeq(keys, ",") {
		if k = strings.TrimSpace(k); k != "" {
			extra[k] = true
		}
	}
	var out []pair
	for _, kv := range env {
		name, value, _ := strings.Cut(kv, "=")
		if value == "" || !config.HasOwnPrefix(name) || !config.Storable(name) {
			continue
		}
		if config.IsSecret(name) || extra[name] {
			out = append(out, pair{name, value})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

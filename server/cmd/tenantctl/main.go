package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/leo/iop/server/internal/app"
	"github.com/leo/iop/server/internal/config"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// tenantctl: small admin CLI used by ops to seed tenants + users.
// Subcommands:
//   tenant create --slug=acme --name="ACME, Inc."
//   tenant list
//   tenant suspend --id=<uuid>
//   tenant resume  --id=<uuid>
//   tenant close   --id=<uuid>
//   user create --email=foo@bar --password=...
//   member join --tenant=<id> --user=<id> --name=... --email=...
//   role grant --tenant=<id> --member=<id> --code=tenant_admin

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx := context.Background()
	a, cleanup, err := app.Build(ctx, cfg)
	if err != nil {
		log.Fatalf("app build: %v", err)
	}
	defer cleanup()

	switch os.Args[1] {
	case "tenant":
		runTenant(ctx, a)
	case "user":
		runUser(ctx, a)
	case "member":
		runMember(ctx, a)
	case "role":
		runRole(ctx, a)
	default:
		usage()
	}
}

func usage() {
	fmt.Println(`tenantctl <verb> <subverb> [flags]

tenant create  --slug   --name
tenant list
tenant suspend --id     [--reason]
tenant resume  --id
tenant close   --id

user create    --email  --password

member join    --tenant --user --name --email [--dept] [--title]

role grant     --tenant --member --code`)
	os.Exit(2)
}

func runTenant(ctx context.Context, a *app.App) {
	if len(os.Args) < 3 {
		usage()
	}
	fs := flag.NewFlagSet("tenant", flag.ExitOnError)
	switch os.Args[2] {
	case "create":
		slug := fs.String("slug", "", "")
		name := fs.String("name", "", "")
		fs.Parse(os.Args[3:])
		t, err := a.Tenancy.CreateTenant(ctx, tenancy.CreateTenantCmd{Slug: *slug, Name: *name})
		dieOnErr(err)
		printJSON(t)
	case "list":
		ts, err := a.Tenancy.ListActiveTenants(ctx, kernel.Pagination{Page: 1, PageSize: 100})
		dieOnErr(err)
		printJSON(ts)
	case "suspend":
		id := fs.String("id", "", "")
		reason := fs.String("reason", "", "")
		fs.Parse(os.Args[3:])
		tid, _ := kernel.ParseID(*id)
		dieOnErr(a.Tenancy.SuspendTenant(ctx, tid, *reason))
	case "resume":
		id := fs.String("id", "", "")
		fs.Parse(os.Args[3:])
		tid, _ := kernel.ParseID(*id)
		dieOnErr(a.Tenancy.ResumeTenant(ctx, tid))
	case "close":
		id := fs.String("id", "", "")
		fs.Parse(os.Args[3:])
		tid, _ := kernel.ParseID(*id)
		dieOnErr(a.Tenancy.CloseTenant(ctx, tid))
	default:
		usage()
	}
}

func runUser(ctx context.Context, a *app.App) {
	if len(os.Args) < 3 {
		usage()
	}
	fs := flag.NewFlagSet("user", flag.ExitOnError)
	switch os.Args[2] {
	case "create":
		username := fs.String("username", "", "")
		email := fs.String("email", "", "")
		password := fs.String("password", "", "")
		fs.Parse(os.Args[3:])
		u, err := a.IAM.RegisterUser(ctx, iam.RegisterCmd{
			Username: *username, Email: *email, Password: *password,
		})
		dieOnErr(err)
		printJSON(u)
	default:
		usage()
	}
}

func runMember(ctx context.Context, a *app.App) {
	if len(os.Args) < 3 {
		usage()
	}
	fs := flag.NewFlagSet("member", flag.ExitOnError)
	switch os.Args[2] {
	case "join":
		tenant := fs.String("tenant", "", "")
		user := fs.String("user", "", "")
		name := fs.String("name", "", "")
		email := fs.String("email", "", "")
		dept := fs.String("dept", "", "")
		title := fs.String("title", "", "")
		fs.Parse(os.Args[3:])
		tid, _ := kernel.ParseID(*tenant)
		uid, _ := kernel.ParseID(*user)
		m, err := a.Tenancy.JoinMember(ctx, a.Pool, tenancy.JoinMemberCmd{
			PlatformUserID: uid, TenantID: tid,
			DisplayName: *name, Email: *email,
			Department: *dept, Title: *title,
		})
		dieOnErr(err)
		printJSON(m)
	default:
		usage()
	}
}

func runRole(ctx context.Context, a *app.App) {
	if len(os.Args) < 3 {
		usage()
	}
	fs := flag.NewFlagSet("role", flag.ExitOnError)
	switch os.Args[2] {
	case "grant":
		tenant := fs.String("tenant", "", "")
		member := fs.String("member", "", "")
		code := fs.String("code", "", "")
		fs.Parse(os.Args[3:])
		tid, _ := kernel.ParseID(*tenant)
		mid, _ := kernel.ParseID(*member)
		dieOnErr(a.IAM.GrantRoleByCode(ctx, mid, tid, *code))
	default:
		usage()
	}
}

func dieOnErr(err error) {
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}

func printJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

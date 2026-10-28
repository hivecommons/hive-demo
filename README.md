# Try Hive on an existing repo

A GitHub template for trying [Hive](https://github.com/hivecommons/hive). Make your own copy, then
follow this README to take it up the ACMM levels. It currently covers L1 → L2 → L3, ending with you
reviewing and accepting a pull request that a Hive agent wrote and held for you.

The repo stands in for one your team already has: a small Go service (a bookmarks API) with working
CI, decent but imperfect, and nothing Hive-specific in it. The walkthrough shows, by hand, each change
a team makes to its own repo to work with Hive, and the level at which it becomes necessary.

> Status: work in progress. The script that runs Hive against your copy is not written yet.

## What's in the repo

| Path | What it is |
|---|---|
| `main.go`, `handlers/`, `store/` | The sample service. Some of it is tested, some is not, and it has a bug or two for Hive to find. |
| `docs/API.md` | The service's API docs. |
| `.github/workflows/ci.yml` | CI: `go vet` and `go test` on every PR. |
| `setup/` | Files you apply by hand during the walkthrough: the `hold-gate` workflow and two rulesets. |
| `hive/` | Runs Hive in Docker against your copy: `compose.yaml` and Hive's starting config, `hive.yaml`. |

Don't fix the service before you start: its gaps are what Hive's agents find.

## Getting started

Hive acts on GitHub as a **GitHub App** that you create once and install on your copy. You run the
admin steps as yourself, with your normal `gh` login.

### Why an App

- **The agent's PR is not yours.** Hive opens PRs as `<app-slug>[bot]`. GitHub never lets an author
  approve their own PR, so if Hive used your token, "a human accepts the agent's PR" would become "you
  merge your own PR". With the App, the bot opens the PR and you review, approve and merge it.
- **No second account.** No signup, email or 2FA to manage.
- **Narrow access.** The App is installed on your copy only, with only the permissions you give it
  in step 3. Hive mints its own short-lived tokens from the private key; there is no long-lived bot
  token.
- **It cannot touch CI.** The App has no Workflows permission, so GitHub refuses any push from it
  that touches `.github/workflows/`. That keeps it from editing the `hold-gate` workflow in its own PR
  to make the check pass. The cost is that the quality agent cannot change CI; the sample app ships
  working CI, so it does not need to.

### 1. Give your `gh` login the admin scopes

```sh
gh auth refresh --hostname github.com --scopes repo,workflow,delete_repo
gh auth status                 # check the scopes line
```

- `repo`: create your copy, labels, rulesets.
- `workflow`: push the `hold-gate` workflow over HTTPS.
- `delete_repo`: delete your copy when you are done.

If your copy will live in an organisation, you must be able to create repositories and GitHub Apps
in it.

### 2. Make your copy

Click **Use this template** on GitHub, or:

```sh
gh repo create <you>/hive-demo --template hivecommons/hive-demo --public --clone
cd hive-demo
```

Make it public: rulesets on private repos need a paid plan. Run every command below from inside
your clone; `gh` works out the repo from there.

### 3. Create the App

Signed in to GitHub as the owner of your copy, go to **Settings → Developer
settings → GitHub Apps → New GitHub App** (<https://github.com/settings/apps/new>). For an
organisation, use the organisation's **Settings → Developer settings → GitHub Apps** instead.

Fill in the form, top to bottom:

- **GitHub App name**: anything unique across GitHub, e.g. `<you> Hive Demo`. The App's slug (used
  in its URLs, and as `HIVE_GITHUB_APP_SLUG` later) is this name in lower case with hyphens.
- **Description**: optional.
- **Homepage URL**: anything, e.g. `https://github.com/hivecommons/hive`.
- **Identifying and authorizing users**: leave **Redirect URI** empty and the checkboxes as they
  are (Device Flow and the OAuth-during-installation request both off). Hive does not sign users in
  through the App.
- **Post installation**: leave **Setup URL** empty and **Redirect on update** unticked.
- **Webhook**: untick **Active**. The **Webhook URL** field then stops being required. Hive polls
  GitHub, and finds the installation by itself.
- **Permissions → Repository permissions**: click the section to expand it, then set:

  | Permission | Access | Why |
  |---|---|---|
  | Contents | Read and write | Clone, branch and push the L3 PR |
  | Issues | Read and write | The advisory tracking issue, labels |
  | Pull requests | Read and write | Reviews at L2, the PR at L3 |
  | Actions | Read-only | Read CI results |
  | Checks | Read-only | Read CI results |
  | Commit statuses | Read-only | Read CI results |
  | Metadata | Read-only | Set automatically; required by GitHub |
  | Workflows | **No access** | See "Why an App" |

  Leave everything else at **No access**, including **Organization permissions** and **Account
  permissions**.
- **Subscribe to events**: leave all unticked (there is no webhook to send them to).
- **Where can this GitHub App be installed?**: **Only on this account**.

Click **Create GitHub App**. The App's settings open, with a sidebar (**General**, **Permissions &
events**, **Credentials**, **Optional features**, **Advanced**, **Install App**, **Public page**):

1. On **General**, note the **App ID**. The slug is the last part of the App's public URL
   (`https://github.com/apps/<slug>`); **Public page** in the sidebar opens it.
2. On **Credentials → Key pairs**, click **New key**. The browser downloads a `.pem` private key.
   Move it into place:

   ```sh
   mkdir -p secrets
   mv ~/Downloads/<downloaded-file>.pem secrets/gh-app-key.pem
   chmod 600 secrets/gh-app-key.pem
   ```

   Run this in your clone. `secrets/` is git-ignored. Anyone with this file can act as the App.

You install the App on your repo later, as one of the demo steps.

### 4. Set the environment variables

Set these in the shell you run Hive from. Nothing stores them; set them again in a new shell.

```sh
export HIVE_REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)   # your copy, e.g. you/hive-demo
export HIVE_GITHUB_APP_ID=<App ID>               # from step 3
export HIVE_GITHUB_APP_SLUG=<slug>               # from step 3
export HIVE_DASHBOARD_TOKEN=$(openssl rand -hex 24)
```

`HIVE_DASHBOARD_TOKEN` is the password for the Hive dashboard and API; holding it is owner access.
Any long random string works. Keep it if you want to reuse a running Hive from another shell.

### Without an App

Hive also accepts a classic personal access token (`github.token: ${HIVE_GITHUB_TOKEN}` in
`hive.yaml`). To keep the agent's PRs separate from yours, the token must belong to a second GitHub
account (a machine user), invited to the repo as a collaborator with **Write**. Give it the `repo`
scope only; leaving out `workflow` has the same effect as the App's missing Workflows permission.
Fine-grained tokens cannot reach a repo the account is only a collaborator on.

## Making the repo Hive-ready

Each step is a change a team would make by hand to its own repo, done at the level where Hive starts
to need it. In `gh api` paths, `{owner}/{repo}` is filled in by `gh` from your clone; type it as
shown.

### Before starting Hive (L1): install the App

Hive writes to the repo from its first boot: even at L1 it opens a "🐝 Hive Advisory Report" tracking
issue, and at L2 the reviewer comments on PRs. It does all of this as the App, so the App has to be
installed on the repo.

Open the App's install page (or **Install App** in the App's settings sidebar):

```sh
xdg-open "https://github.com/apps/$HIVE_GITHUB_APP_SLUG/installations/new"
```

Choose **Only select repositories**, pick your copy, and click **Install**.

If you have installed the App before (on an earlier copy), the same page opens the installation's
**Configure** screen instead. Add the new repo under **Repository access** and save.

Nothing to configure in Hive for this: it finds the installation itself.

### Start Hive (L1)

Hive runs in a single container, from [`hive/compose.yaml`](hive/compose.yaml), using the variables
from step 4 and the App key in `secrets/`. It needs Docker with Compose, or Podman with
`docker compose`.

```sh
docker compose -f hive/compose.yaml up -d
docker compose -f hive/compose.yaml ps      # wait for "healthy", up to two minutes on first start
```

Open <http://127.0.0.1:3001> and sign in with `$HIVE_DASHBOARD_TOKEN`.

Hive starts at L1 with two agents, **brainstorm** and **guide**, both on Claude
([`hive/hive.yaml`](hive/hive.yaml)). Sign in to Claude once: open an agent in the dashboard and
follow its sign-in prompt. The sign-in is kept in Hive's data volume and shared by every agent.

What to expect:

- Hive opens a "🐝 Hive Advisory Report" tracking issue on your repo within a few minutes.
- Only **brainstorm** starts paused; it runs when you start an inception. Every other agent runs on
  its own schedule and uses your Claude plan. Pause the ones you are not watching.
- To run an agent now, resume it, then kick it (kicking a paused agent fails).

Stop and reset:

```sh
docker compose -f hive/compose.yaml down       # stop; keeps Hive's state and the Claude sign-in
docker compose -f hive/compose.yaml down -v    # stop and delete everything Hive kept
```

Hive reads `hive/hive.yaml` only on first start, then keeps its own copy and updates it as you
promote. To start from the file again, use `down -v` (you sign in to Claude again).

On first start Hive takes ownership of `secrets/gh-app-key.pem` and makes it readable only by its
own user, so you may not be able to read the file afterwards. Keep the downloaded original if you
need it.

### Promoting

Promote in the dashboard's **Governor** settings, or with the API:

```sh
curl -X POST -H "Authorization: Bearer $HIVE_DASHBOARD_TOKEN" http://127.0.0.1:3001/api/packs/2/apply
```

Use `3` for L3. Each level adds its agents to the ones already running. Make the repo changes for a
level, below, before you promote to it.

### Before promoting to L2: send changes through pull requests

L2 adds advisory agents. Nothing they do needs new GitHub access: findings go to the Hive dashboard and
to the tracking issue, the reviewer posts comment-only reviews (never approves or merges), and Hive
creates its own triage labels, all as the App.

But the reviewer works from the open PR queue, so it only sees changes that arrive as pull requests.
The **approval** ruleset ([`setup/rulesets/approval.json`](setup/rulesets/approval.json)) makes that the rule for the default
branch: a pull request with one approving review, with a pull-request bypass for repository admins so
you can still merge your own PRs (`gh pr merge --admin`). The App cannot approve its own PRs, so
nothing it writes reaches the default branch without your approval.

```sh
gh api -X POST "repos/{owner}/{repo}/rulesets" --input setup/rulesets/approval.json
```

In the UI: **Settings → Rules → Rulesets → New ruleset → New branch ruleset**, target the default
branch, enable **Require a pull request before merging** with 1 required approval, and add
**Repository admin** to the bypass list.

### Before promoting to L3: make `hold` enforceable

At L3 the quality agent opens PRs, and labels every one `hold`, `quality` and `testing`. GitHub ignores
labels when merging, so a held PR could still be merged by anyone with the approval. These three steps
turn `hold` into a gate.

**1. Create the labels.** Adding a label that does not exist can fail the agent's PR.

```sh
gh label create hold    --color b60205 --description "Held for human review; blocks merge"
gh label create quality --color 0e8a16
gh label create testing --color 1d76db
```

**2. Add the `hold-gate` check through a pull request.** [`setup/hold-gate.yml`](setup/hold-gate.yml)
fails while a PR carries `hold` and re-runs when the label is removed. It sits in `setup/` so it does
nothing until you move it into `.github/workflows/`, the way any change reaches the repo:

```sh
git switch -c add-hold-gate
git mv setup/hold-gate.yml .github/workflows/hold-gate.yml
git commit -m "Add hold-gate check"
git push -u origin add-hold-gate
gh pr create --fill
gh pr merge --squash --delete-branch --admin    # --admin: the L2 approval ruleset's bypass
git switch main && git pull
```

Do this before step 3: the rulesets UI only offers checks that have already run on the repo.

**3. Add the hold-gate ruleset.** ([`setup/rulesets/hold-gate.json`](setup/rulesets/hold-gate.json)) The `hold-gate` check from
GitHub Actions must pass on the default branch. No bypass, for anyone, including admins.

```sh
gh api -X POST "repos/{owner}/{repo}/rulesets" --input setup/rulesets/hold-gate.json
```

In the UI: **Settings → Rules → Rulesets → New ruleset → New branch ruleset**, target the default
branch, enable **Require status checks to pass**, add `hold-gate` with source GitHub Actions, and
leave the bypass list empty.

This has to be its own ruleset, separate from the approval one: a ruleset's bypass applies to all of
its rules, so putting `hold-gate` in the approval ruleset would let admins skip it too.

Check it worked: a PR labelled `hold` shows `hold-gate` failing and the Merge button disabled, for you
as well. Remove the label and the check goes green.

## Starting again

Reset Hive, delete your copy, then make a new one (step 2) and add it to the App's installation
(L1):

```sh
docker compose -f hive/compose.yaml down -v
gh repo delete <you>/hive-demo --yes
```

Every issue and PR goes with it. To remove the App too, delete it from **Advanced** in its settings
sidebar, and delete `secrets/gh-app-key.pem`.

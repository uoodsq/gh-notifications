# gh-notifications :no_bell:

A [GitHub CLI][1] extension that marks notifications as done for you.

[1]: https://github.com/cli/cli

## Why?

I get a ton of notifications in my GitHub Inbox (on the order of hundreds/day), mostly from repos I don't actually care about, but am subscribed to anyway because of org and group membership.

GitHub's notifications UI is not built well for dismissing en masse, so I wrote this to help find signal in the noise.

## Installation

`gh-notifications` has no dependencies.  Just install the extension! :rocket:

```shell
gh extension install uoodsq/gh-notifications
```

## Usage

This extension works by syncing notification data from GitHub to disk.  Pull down notifications using `gh notifications sync`:

```shell
gh notifications sync
...
╭─────────────────────┬────────────┬────────────────────────────╮
│                REPO │ ID         │ TITLE                      │
├─────────────────────┼────────────┼────────────────────────────┤
│ acme/important-repo │ 1234567890 │ This is a notification!    │
├─────────────────────┼────────────┼────────────────────────────┤
│     acme/noisy-spam │ 2837482910 │ Review this PR!            │
│                     │ 2887828492 │ Check out this issue!      │
│                     │ 4872983794 │ Grok this discussion!      │
│                     │ 9037742732 │ More notification noise... │
│                     │ 9051534601 │ Need your attention!       │
╰─────────────────────┴────────────┴────────────────────────────╯
```

You can mark a repo as ignored.  This will dismiss all active notifications for that repo, as well as any future notifications pulled via `gh notifications sync`:

```shell
gh notifications ignore acme/noisy-spam
2024/02/09 20:19:13 marking 'Review this PR!' done
2024/02/09 20:19:13 marking 'Check out this issue!' done
2024/02/09 20:19:13 marking 'Grok this discussion!' done
2024/02/09 20:19:14 marking 'More notification noise...' done
2024/02/09 20:19:14 marking 'Need your attention!' done
╭────────────────────────────────────╮
│ IGNORED REPO                       │
├────────────────────────────────────┤
│ acme/noisy-spam                    │
╰────────────────────────────────────╯
```

You can also open a notification's subject by clicking on its ID (if your terminal emulator supports hyperlinks), and dismiss the notification in the CLI:

```shell
gh notifications done 2837482910
2024/02/09 20:19:20 marking 'Review this PR!' done
╭─────────────────────┬────────────┬────────────────────────────╮
│                REPO │ ID         │ TITLE                      │
├─────────────────────┼────────────┼────────────────────────────┤
│ acme/important-repo │ 1234567890 │ This is a notification!    │
├─────────────────────┼────────────┼────────────────────────────┤
│     acme/noisy-spam │ 2887828492 │ Check out this issue!      │
│                     │ 4872983794 │ Grok this discussion!      │
│                     │ 9037742732 │ More notification noise... │
│                     │ 9051534601 │ Need your attention!       │
╰─────────────────────┴────────────┴────────────────────────────╯
```

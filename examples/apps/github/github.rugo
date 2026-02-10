#!/usr/bin/env rugo
#
# GitHub CLI — a showcase for the rugh library
#
# Usage:
#   export GITHUB_TOKEN=ghp_xxxx
#   rugo run github.rg me
#   rugo run github.rg repos
#   rugo run github.rg repos rubiojr
#   rugo run github.rg repo rubiojr rugo
#   rugo run github.rg issues rubiojr rugo
#   rugo run github.rg issue rubiojr rugo 42
#
use "cli"
use "conv"
use "color"

require "github.com/rubiojr/rugh" with client, repo, issue, user

cli.name "github"
cli.version "0.1.0"
cli.about "GitHub CLI powered by rugh"

cli.cmd "me", "Show authenticated user info"
cli.cmd "repos", "List repositories (yours, by username)"
cli.cmd "repo", "Show repository details"
cli.cmd "issues", "List issues for a repository"
cli.cmd "issue", "Show issue details"

cli.run

def me(args)
  gh = client.from_env()
  me = user.me(gh)
  puts color.bold(conv.to_s(me["login"]))
  puts conv.to_s(me["name"])
  puts color.dim(conv.to_s(me["bio"]))
  puts ""
  puts color.cyan(conv.to_s(me["html_url"]))
  puts color.gray("Repos: " + conv.to_s(me["public_repos"]) + " | Followers: " + conv.to_s(me["followers"]))
end

def repos(args)
  gh = client.from_env()
 
  repos = nil
  if len(args) > 0
    repos = repo.list_for(gh, args[0])
    puts color.bold("Repos for " + args[0])
  else
    repos = repo.list(gh)
    puts color.bold("Your repositories")
  end
  puts ""
  puts gh
  puts "foo"

  for i, r in repos
    name = conv.to_s(r["full_name"])
    desc = conv.to_s(r["description"])
    stars = conv.to_s(r["stargazers_count"])
    lang = conv.to_s(r["language"])

    puts color.bold(name) + "  " + color.yellow("★ " + stars)
    if desc != "" && desc != "nil"
      puts color.dim("  " + desc)
    end
    if lang != "" && lang != "nil"
      puts color.cyan("  " + lang)
    end
    puts ""
  end
end

def repo(args)
  if len(args) < 2
    puts "Usage: github repo <owner> <name>"
    return nil
  end
  gh = client.from_env()
  r = repo.get(gh, args[0], args[1])

  puts color.bold(conv.to_s(r["full_name"]))
  desc = conv.to_s(r["description"])
  if desc != "" && desc != "nil"
    puts color.dim(desc)
  end
  puts ""
  puts color.cyan(conv.to_s(r["html_url"]))
  puts ""
  puts "Stars: " + color.yellow(conv.to_s(r["stargazers_count"]))
  puts "Forks: " + conv.to_s(r["forks_count"])
  puts "Language: " + conv.to_s(r["language"])
  puts "Default branch: " + conv.to_s(r["default_branch"])
end

def issues(args)
  if len(args) < 2
    puts "Usage: github issues <owner> <name>"
    return nil
  end
  gh = client.from_env()
  issues = issue.list(gh, args[0], args[1])

  puts color.bold("Issues for " + args[0] + "/" + args[1])
  puts ""

  for i, iss in issues
    num = conv.to_s(iss["number"])
    title = conv.to_s(iss["title"])
    author = conv.to_s(iss["user"]["login"])
    comments = conv.to_s(iss["comments"])

    puts color.green("#" + num) + " " + color.bold(title)
    puts color.gray("  by " + author + " | " + comments + " comments")
    puts ""
  end
end

def issue(args)
  if len(args) < 3
    puts "Usage: github issue <owner> <name> <number>"
    return nil
  end
  gh = client.from_env()
  iss = issue.get(gh, args[0], args[1], conv.to_i(args[2]))

  num = conv.to_s(iss["number"])
  title = conv.to_s(iss["title"])
  state = conv.to_s(iss["state"])
  author = conv.to_s(iss["user"]["login"])
  body = conv.to_s(iss["body"])

  puts color.green("#" + num) + " " + color.bold(title)
  if state == "open"
    puts color.green("● open")
  else
    puts color.red("● closed")
  end
  puts color.gray("by " + author)
  puts ""
  if body != "" && body != "nil"
    puts body
  end
end

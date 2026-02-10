#!/usr/bin/env rugo
#
# Hacker News CLI client
#
# A terminal client for Hacker News that showcases all Rugo modules:
#   cli   — command routing, flags, help
#   http  — fetching from HN Firebase API
#   json  — parsing API responses
#   conv  — type conversions for display
#   str   — string manipulation
#   color — colorized terminal output
#
# Usage:
#   rugo run hackernews.rg top
#   rugo run hackernews.rg top -n 5
#   rugo run hackernews.rg new
#   rugo run hackernews.rg best --count 20
#   rugo run hackernews.rg show 12345678
#

use "cli"
use "http"
use "json"
use "conv"
use "str"
use "color"
use "os"

cli.name "hn"
cli.version "0.1.0"
cli.about "Hacker News in your terminal"

cli.cmd "top", "Show top stories"
cli.flag "top", "count", "n", "Number of stories to show", "10"

cli.cmd "new", "Show newest stories"
cli.flag "new", "count", "n", "Number of stories to show", "10"

cli.cmd "best", "Show best stories"
cli.flag "best", "count", "n", "Number of stories to show", "10"

cli.cmd "show", "Show story details by ID"

cli.run

# --- Helpers ---

def fetch_item(id)
  url = "https://hacker-news.firebaseio.com/v0/item/#{id}.json"
  body = http.get(url).body
  return json.parse(body)
end

def current_time()
  return conv.to_i(str.trim(`date +%s`))
end

def time_ago(ts, now)
  diff = now - ts

  if diff < 60
    return "just now"
  end
  if diff < 3600
    mins = conv.to_s(diff / 60)
    return "#{mins}m ago"
  end
  if diff < 86400
    hours = conv.to_s(diff / 3600)
    return "#{hours}h ago"
  end
  days = conv.to_s(diff / 86400)
  return "#{days}d ago"
end

def print_story(rank, item, now)
  title = conv.to_s(item["title"])
  score = conv.to_s(item["score"])
  author = conv.to_s(item["by"])
  comments = 0
  if item["descendants"] != nil
    comments = item["descendants"]
  end

  rank_str = conv.to_s(rank)
  puts color.bold("#{rank_str}. #{title}")

  url = item["url"]
  if url != nil
    host = extract_host(conv.to_s(url))
    puts color.cyan("   #{url}")
    puts color.dim("   (#{host})")
  end

  age = time_ago(item["time"], now)
  comment_str = conv.to_s(comments)
  puts color.gray("   #{score} points by #{author} | #{age} | #{comment_str} comments")
  puts ""
end

def extract_host(url)
  # Strip protocol
  stripped = url
  if str.starts_with(url, "https://")
    stripped = str.replace(url, "https://", "")
  elsif str.starts_with(url, "http://")
    stripped = str.replace(url, "http://", "")
  end
  # Take everything before the first /
  parts = str.split(stripped, "/")
  return parts[0]
end

def show_stories(endpoint, label, emoji)
  count = conv.to_i(cli.get("count"))
  now = current_time()
  puts color.bold("#{emoji} #{label}")
  puts ""

  body = http.get(endpoint).body
  ids = json.parse(body)
  ids = ids[0, count]

  # Fetch all items concurrently
  tasks = []
  for i, id in ids
    t = spawn
      fetch_item(id)
    end
    tasks = append(tasks, t)
  end

  # Print in order
  for i, t in tasks
    print_story(i + 1, t.value, now)
  end
end

# --- Commands ---

def top(args)
  show_stories("https://hacker-news.firebaseio.com/v0/topstories.json", "Top Stories", "🔥")
end

def new(args)
  show_stories("https://hacker-news.firebaseio.com/v0/newstories.json", "New Stories", "🆕")
end

def best(args)
  show_stories("https://hacker-news.firebaseio.com/v0/beststories.json", "Best Stories", "⭐")
end

def show(args)
  if len(args) == 0
    puts color.red("Error: provide a story ID")
    puts "Usage: hn show <id>"
    os.exit(1)
  end

  id = args[0]
  item = fetch_item(id)

  if item == nil
    puts color.red("Story not found: #{id}")
    os.exit(1)
  end

  # Header
  puts color.bold(conv.to_s(item["title"]))
  puts ""

  url = item["url"]
  if url != nil
    puts color.cyan(conv.to_s(url))
    puts ""
  end

  score = conv.to_s(item["score"])
  author = conv.to_s(item["by"])
  age = time_ago(item["time"], current_time())
  comments = 0
  if item["descendants"] != nil
    comments = item["descendants"]
  end
  comment_str = conv.to_s(comments)

  puts color.yellow("#{score} points") + " by " + color.green(author) + " | " + color.gray(age)
  puts color.gray("#{comment_str} comments")

  # Show text if it's an Ask HN / Show HN with body
  if item["text"] != nil
    puts ""
    puts conv.to_s(item["text"])
  end
end

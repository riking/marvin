

# Factoids

# Standard Lua interfaces

## `base` module

```lua
-- Appends the arguments to the output.
print(Stringish [, ...])
-- Prints a table in the output.
ptable(Table)
-- Exits the factoid, appending the provided item to the output.
return Stringish

-- Throws the 2nd argument as an error if the 1st argument is false or nil.
-- Returns all arguments unchanged if the 1st argument is truthy.
assert(Any, String, ...) -> ...

-- Throws the provided argument as an error.
error(Any)

-- See https://www.lua.org/pil/14.3.html
getfenv(Function)
setfenv(Function, Table)
-- See http://lua-users.org/wiki/MetamethodsTutorial
getmetatable(Table) -> Table or Nil
setmetatable(Table, Table mt)

load(Function reader [, String name]) -- (Function) or (Nil, String error)
loadstring(String source [, String name]) -- (Function) or (Nil, String error)

-- for k, v in pairs(t) do ... end
-- for i, v in ipairs(t) do ... end
-- ipairs only iterates numeric indices, while pairs will iterate all indices.
-- next is the internal iterator for pairs().
pairs(Table)
ipairs(Table)
next(Table [, Number index)

-- If the provided function throws an error, pcall will stop and return (false, error).
-- xpcall will call the onerror function.
-- If the function executes without errors, the functions return true followed by the return values of the function.
pcall(Function) -> (Bool okay, ...)
xpcall(Function f, Function onerror) -> (Bool okay, ...)

-- Check equality, table get, and table set, ignoring metamethods.
rawequal(Any, Any) -> Bool
rawget(Table, index) -> Any
rawset(Table, index, value)

-- If index is a number, returns all arguments after argument number index.
-- Otherwise, index must be the string "#", and select returns the total number of extra arguments it received.
select(index, ...) -> ...

-- Converts the argument into a number. Returns nil if not convertible.
tonumber(String e, Number base) -> Number or Nil
tonumber(Number e) -> Number
-- Converts input into a string, possibly calling the __tostring metamethod.
tostring(Any) -> String

-- returns the type of its argument as a string.
type(Any) -> String

-- Unpacks the provided list into a series of return values.
-- The function is similar to
--   return list[i], list[i+1], list[i+2], ..., list[j]
-- except that code can only be written for a fixed number of elements.
-- The default values for i and j select the entire list.
unpack(list [, i, j]]) -> ...

-- These functions throw a forbidden function error when called.
collectgarbage() dofile() loadfile() _printregs()
```

## `table`, `string`, `math` modules

String functions: https://www.lua.org/manual/5.1/manual.html#5.4  
Table functions: https://www.lua.org/manual/5.1/manual.html#5.5  
Math functions: https://www.lua.org/manual/5.1/manual.html#5.6

## `debug` module

https://www.lua.org/manual/5.1/manual.html#5.9

`debug.debug()`, hook functions, and `debug.getregistry()` are not available.

# Additional Lua interfaces

## `bit` module

```lua
-- clamps the input to an int32
bit.tobit(Number) -> Number
-- converts input to a hex string
bit.tohex(Number) -> String

-- Bitwise not, and, or, xor, left shift.
bit.bnot(Number) -> Number
bit.band(Number, Number) -> Number
bit.bor(Number, Number) -> Number
bit.bxor(Number, Number) -> Number
bit.lshift(Number, Number num_bits) -> Number
-- rshift performs a 0-extended right shift, while arshift performs a sign-extended right shift
bit.rshift(Number, Number num_bits) -> Number
bit.arshift(Number, Number num_bits) -> Number
```

## `bot` module

```lua
bot.now() -> Number -- Unix timestamp, see also time module

-- Performs URL encoding and decoding, useful for passing to other factoids
bot.uriencode(String) -> String
bot.uridecode(String) -> String

-- Converts Slack entities into fallback test
bot.unescape(String) -> String

-- Converts a Unicode codepoint into a string
bot.unichr(Number) -> String

-- Paste returns a URL that will respond with the arguments when fetched.
-- Shortlink returns a URL that will redirect to the arguments when fetched.
bot.paste(String) -> String
bot.shortlink(String) -> String
```

Example:

```lua
local test_content = "123456789"
local paste_url = bot.paste(test_content)
local resp, err = requests.get(paste_url)
print(test_content == resp:text())
-- true
```

## `corpus` module

See https://github.com/dariusk/corpora

Example:

```lua
-- https://github.com/dariusk/corpora/blob/master/data/technology/computer_sciences.json
local example_dataname = "technology/computer_sciences"

ptable(corpus.info[example_dataname])
-- field: "computer_sciences"
-- description: "names of technologies related to computer science"
-- source: ""
-- length: 197

print(corpus[example_dataname][2])
-- ActionScript
```

## `intra` module

Slightly broken, won't bother documenting for now

## `json` module

Overview

```lua
json.load(String) -> Any
  -- aliases: json.decode json.parse
json.dump(Any) -> String
  -- aliases: json.encode
json.null -> Userdata(JsonNull)
  -- aliases: _G.jsonNull
json.mt_isarray -> Table
json.mt_isobject -> Table
```

#### Conversion between JSON and Lua

Bools, numbers, and strings are round-tripped through Lua and JSON without issue.

Any `null`s in JSON will become the `jsonNull` object in Lua to avoid ambiguity with missing indices.

Any `nil`s in Lua will become `null`s in JSON, as well as the `jsonNull` object.

JSON arrays and objects will both become Lua tables, but arrays will have numeric indices, and have their metatable
set to `json.mt_isarray`, while JSON objects will have string indices, and have their metatable set to `json.mt_isobject`.

Lua tables are inspected to see whether they should be handled as arrays or objects.

 - If the table has any keys that are not Numbers or Strings, an error is raised.
 - Any table with numeric indices greater than 1 million will be marshalled as an object with numeric string keys instead.
 - If the table's metatable is `json.mt_isobject` or `json.mt_isarray`, it is marshalled as that respective type.
 - If the table has any string indices, it is marshalled as an object with numeric indices converted into their respective numeric string.
 - If the table has any numeric indices, it is marshalled as an array of size `#t`.
 - Otherwise, it is marshalled as an empty object.

Threads, functions, and channels cannot be encoded and will raise an error if attempted.
Only userdata objects that opt-in to JSON marshalling can be encoded. Other userdata objects will raise an error containing their Go-side type.

Recursive tables will blow the call stack and crash the interpreter.

## `requests` module

The requests module provides access to HTTP requests.

```lua
-- get, head, delete, trace
resp, err = requests.get(url, headers, options)
-- post, put, options, patch
resp, err = requests.post(url, headers, data, options)
resp, err = requests.request(url, headers, data, method, options)

-- url: String
-- headers: Nil or Table
-- data: Nil or String or Table (x-www-urlencoded)
-- method: GET, HEAD, DELETE, TRACE, POST, PUT, OPTIONS, PATCH
-- options: Nil or Table
```

No options are currently defined.

The function returns as soon as the **headers are received**.

#### LResponse

A LResponse is userdata with __index fields.
```
resp.headers -> Table -- map of header name to header value
resp.statuscode -> Number -- e.g. 200
resp.status -> String -- e.g. "200 OK"
resp.proto -> String -- e.g. "HTTP/1.1"

-- The following two functions consume the response body, so may take time to execute.
-- They cache their result, so can be called multiple times.
resp:text() -> String
resp:json() -> Any -- see json module documentation
```

## `time` module

The time module provides access to time formatting functions from Lua.

```lua
time.rfc3339 -> String -- formatting constant
time.now() -> Time -- returns the current time
-- creates a new Time from a unix timestamp
time.fromunix(sec, nsec=0) -> Time
-- Parses a time with the given format
-- https://godoc.org/time#pkg-constants
time.parse(String value, format=time.rfc3339) -> (Time, error)
```

Type: Time

```lua
local t = time.now()

t.year -> Number: year number
t.month -> Number
t.day -> Number
t.hour -> Number
t.minute -> Number
t.second -> Number
t.ns -> Number -- nanoseconds into the second
t.tz -> String -- tzinfo-based timezone string (e.g. Europe/Paris)
t.__is_a_time -> Bool -- type indicator
t:format(str) -> String -- formats a time according to https://godoc.org/time#pkg-constants
t:unix() -> Number, Number -- returns the unix time (seconds, nanoseconds)
```

## `slack` object

`slack.channels` provides a way to get LChannel objects.

```lua
local chs = {slack.channels.general, slack.channels.random}
table.append(chs, slack.channels["G2WLZU48P"]) -- channel ID for #marvin-dev

-- You cannot access the DM channel of other users (unless admin)
local not_permitted = slack.channels.D2XTZTJ22
print(not_permitted == nil) -- true
-- You also cannot access the channel object for private groups you are not a member of.
```

```lua
return slack.archive(channel.id, "1490307275.293641")
-- https://42schoolusa.slack.com/archives/G2WLZU48P/p1490307275293641
```

## LChannel

Several fields of LChannel are not available when the channel is a 1-on-1 DM. Make sure to inspect the `type` before using those.

LChannels can be compared for equality.

```lua
local ch = channel -- the channel the factoid was sent in
local ch2 = slack.channels.general

ch.id -> String -- C1G4AJ96D
ch.type -> String -- "public"
  -- "public", "group", "mpim", or "im"
ch.name -> String -- "general", "[IM with UYOURUSERID]"
-- List of every user in the channel. Warning: may time out on large channels
ch.users -> Table<LUser>
-- Format the channel for display in a response
-- <#C1G4AJ96D|#general>
tostring(ch) -> String
ch.mention -> String

-- Only for non-IM channels
ch.creator -> LUser -- @gaetan
ch.topic, ch.purpose -> String
ch.topic_changed, ch.purpose_changed -> Number -- unix timestamp
ch.topic_user, ch.purpose_user -> LUser -- last set by

-- Only for IM channels
ch.im_other -> LUser -- you
```

## LUser

```lua
local u = user -- user running the factoid

tostring(user) -> "<@USLACKBOT>"
user.id -> "USLACKBOT"

user.is_blacklisted -> Bool
user.is_admin -> Bool
user.is_controller -> Bool
user.deleted -> Bool

user.username -> "slackbot"
user.fname -> String
user.lname -> String
user.name -> String
user.tz -> "America/Los_Angeles"
user.tz_offset -> -28800 -- offset in seconds
user.profile.real -> String
user.profile.first -> String
user.profile.last -> String
user.profile.phone -> String
```

## LFactoid

```lua
local f = factoid.test
f.exists -> Bool
f(...args...) -> String -- runs the factoid after joining arguments as strings
f.src -> String -- alias: f.raw
f.author -> LUser
f.locked -> Bool
f.islocal -> Bool
f.time -> Number -- Unix timestamp
f.created -> String -- Slack archive link
f.data -> FDataMap
```

## LFDataMap

Factoid data is a way to persist state across factoid invocations.

A fdata object acts like a table, but accessing it actually performs function calls - be careful to write back your
modified values when finished.

Concurrent modification is not protected - if two executing factoids mutate the same key, results will be inconsistent.

Written values are persisted to the database every 30 seconds, as well as on shutdown.

The maximum length of a key is 500 bytes. The maximum length of a value is 40 KByte after JSON encoding.

Requesting the length or iterating a fdata object loads **all** content into memory. Avoid this if you're storing a large amount of data across different keys.

The `fdata` object is keyed to the currently executing factoid's name.  
The `fmap` provides access to various global tables shared across factoids.

### Example Usage

Note the following:

 - The 'chamber' table is read, modified, then written. Returned objects are not rechecked for modification at the end of execution, all changes must be explicitly writen back.

https://marvin.riking.org/factoids/_/roulette2

```lua
local d = fdata
function reload()
  -- ...
  d.last = ""
  d.chamber = ch
  d.sb = bul -- setup_bullets
  d.sz = mx -- setup_size
  -- ...
end
function pull()
  local ch = d.chamber
  -- ...
  table.remove(ch, math.random(#ch))
  d.chamber = ch
  d.last = user.id
  -- ...
end
if d.chamber == nil then reload() end
-- ...
```

## Context globals

```lua
-- Arguments to the factoid
argv -> Table<String>
args -> String
-- User, Channel where factoid was run
user -> LUser
channel -> LChannel

slack -> LTeam

-- Name of factoid
factoidname -> String
-- Factoid database object
factoid.example -> LFactoid
factoid[factoidname] -- currently running factoid object

fmap.store_name -> LFDataMap -- Shared factoid datastore
fdata -> LFDataMap -- Private factoid datastore
```

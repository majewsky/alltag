# Alltag - the ADHD-friendly issue tracker

I'm not formally diagnosed with ADHD, but everything I've read about it closely
matches my personal experiences. Alltag is an issue tracker specially suited
for my own needs in order to help me be more productive, and maybe it could
also be of interest to others as well.

Alltag departs from other issue trackers in several significant ways:

- I often notice that I don't add new tasks to my issue tracker as they pop up
  in my head, because I get distracted and forget about it before I get around
  to entering the task. Alltag makes adding new issues as accessible as
  possible by requiring only a one-line summary on a new task. All other
  attributes (due date, priority, recurrence) can be added later during
  *classification*. The start page reminds me whenever there is a task awaiting
  classification.

- Tasks are often tied to locations. Shopping for groceries can only be done in
  the city. Cleaning the home can only be done, well, at home. And so on. This
  relationship can be expressed in most issue trackers in some way, e.g. by
  adding tags to issues. But it requires more clicks than it should. Alltag
  puts this workflow front and center: "I'm in the city right now. Tell me what
  needs to be done here."

- Alltag also adds a unique categorization: Tasks are always labeled as either
  "physical" or "mental". This allows me to switch gears and stay productive
  when my resources on one side are depleted: "I spent the day hiking and now
  I'm exhausted. Give me a mental task where I can sit down while doing it."
  vs. "I spent all day studying and cannot concentrate anymore. Give me a
  physical task that doesn't require too much thinking."

- I find that having to choose tasks from a large backlog leads to choice
  paralysis. I work best when someone else just gives me one task to focus on
  at a time. At work, my manager or a more senior colleague prioritizes my
  backlog, and then I can just grab tasks from the top of the queue one by one.
  At home, Alltag does the same for me. The main UI is not a Kanban board or a
  list. The main UI just asks "Where are you? Do you want a physical or mental
  task?" Then it gives me the most urgent task fitting that choice, taking into
  account both due date and priority of all matching tasks. (That's not to say
  that there is no task list UI. There is, but it's not front and center.)

- *(TODO: not implemented yet)*
  In most issue trackers, the workflow focuses on starting with the big
  picture, then breaking large tasks down into small pieces. This does not
  really work for me: Large tasks are harder to pick up and work on than small
  tasks. Alltag helps with this by emphasizing chaining over decomposition:
  Instead of entering a large task into the tracker, then breaking it down, I
  enter the smallest-possible first step of the task into the backlog. When
  that task is done, Alltag offers me to enter the next step as a new task, and
  so forth.

- In general, the UI is structured around several workflows that are designed
  to be as simple as possible, with not more than a handful of options on each
  screen, to avoid the aforementioned choice paralysis. The more detailed UIs
  are only used for very infrequent tasks like backlog cleanups.

## Linear priority interpolation

To choose "the most urgent task" deterministically, Alltag assigns a
numerical priority value to each task using the following attributes that all
tasks must have:

- an **initial priority** and **final priority** of either *Low* (0), *Normal*
  (1), *High* (2) or *Critical* (3), where the final priority must be higher
  than the initial priority,
- a **start date** (when the task has been entered, or when the task has been
  spawned by a recurrence rule), and a **due date**.

Tasks only show up on the UI after the start date. Until the due date, the
priority then increases linearly from the initial priority to the final
priority. After the due date, the priority continues increasing at the same
rate as before, potentially reaching values of above critical.

# Running Alltag

## Required dependencies

To build:

- [Go 1.12+](https://golang.org)
- [sassc](https://github.com/sass/sassc)

To run:

- [PostgreSQL](https://postgresql.org)
- optionally, an HTTPS server (see below)
- an LDAP server

Alltag exposes its API and UI via HTTP. I highly recommend you reverse-proxy
through a web server like nginx or Apache to add TLS encryption.

If you don't have an LDAP server at hand, may I interest you in another one of
my projects? [Portunus](https://github.com/majewsky/portunus) is a turn-key
solution for everyone who just wants an LDAP server with included admin web
interface, without having to learn the ins and outs of LDAP.

## Building

Build like any other application with a Makefile. (`go get` is not supported.)
The Makefile understands the standard environment variables `PREFIX` and
`DESTDIR`. For instance:

```sh
make && make install PREFIX=/usr/local
```

## Configuration

Once installed, run `alltag` with root privileges. Configuration is passed to
it via the following environment variables:

| Variable | Default | Explanation |
| -------- | ------- | ----------- |
| ALLTAG\_DB\_URI | *(required)* | A [libpq connection URI][pq-uri] that locates the Alltag database. The non-URI "connection string" format is not allowed; it must be a URI. |
| ALLTAG\_LDAP\_URI | *(required)* | Where to reach the LDAP server. The protocol must be either `ldap` or `ldaps`. For protocol `ldap`, StartTLS is required. (If your LDAP does not do TLS, please check back when it can.) |
| ALLTAG\_LDAP\_BIND\_DN | *(required)* | The DN of the service user account that Alltag can bind as to search in the directory. |
| ALLTAG\_LDAP\_BIND\_PASSWORD | *(required)* | The password for that user account. |
| ALLTAG\_LDAP\_SEARCH\_BASE\_DN | *(required)* | Where to search for user accounts. This usually refers to a group of users, e.g. `ou=users,dc=example,dc=com`. |
| ALLTAG\_LDAP\_SEARCH\_FILTER | `(uid=%s)` | Which objects to match on when searching for user accounts in LDAP. The placeholder `%s` will be replaced with the username in question. |
| ALLTAG\_LISTEN\_ADDRESS | `127.0.0.1:8080` | Listen address for the HTTP server exposing Alltag's UI and API. |

Once everything is set up, connect to Alltag via HTTP (either directly or
through a reverse proxy as suggested above). There is no separate login site.
When your browser asks for credentials, enter the name and password for your
user account in LDAP.

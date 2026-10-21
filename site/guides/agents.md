# AI agents

`origins mcp` serves a corpus to an AI agent over the Model Context Protocol (MCP). The
agent can read the tree, ask the same questions as the commands, and propose
hypotheses. It can't change a record, and it can't accept or reject a hypothesis. You
review what it proposes in the same way as any other hypothesis.

## Connecting a client

The agent's client starts `origins mcp` and talks to it on standard input and output.
Most clients take a server as a command and its arguments in their configuration. In
the `mcpServers` form that many clients use, it looks like this:

```json
{
  "mcpServers": {
    "origins": {
      "command": "origins",
      "args": ["mcp", "-C", "/home/rosa/genealogy/rossi", "--as", "assistant"]
    }
  }
}
```

`-C` names the corpus. `--as` names who proposes the hypotheses that the agent writes,
and Origins records that name in each one. Without `--as`, Origins records the name
that the client gives itself.

In Claude Desktop, this goes in the configuration file that **Settings → Developer →
Edit Config** opens, and the server is available after Claude Desktop restarts. In
Claude Code, you add the server with one command:

```bash
claude mcp add origins -- origins mcp -C /home/rosa/genealogy/rossi --as claude
```

If the client doesn't find `origins`, give the full path of the binary as the
command. Each server serves one corpus, so add one server for each corpus, with a
different name.

## Working with an agent

Ask the agent questions about the tree, or give it a task that ends in a proposal. For
example:

- "Who are Giovanni Rossi's great-grandparents, and which of them have no sources?"
- "List the births that `todo` reports without a source, oldest first."
- "Read the baptism of 1842 in battesimi-borgo-san-lorenzo and propose a hypothesis
  for anything it says that the corpus doesn't."

The agent reads through the tools below. Any change it wants to make reaches you as an
open hypothesis, which you test and then accept or reject.

## Tools

The agent can use these tools. Each answers with the JSON that the command's `--json`
gives:

| Tool | Command |
|---|---|
| `index` | `origins index` |
| `read` | one record, as `origins browse` shows it |
| `check` | `origins check`, with `with` for hypotheses |
| `todo` | `origins todo` |
| `ancestors`, `descendants`, `kinship` | the commands of the same names |
| `log`, `whence`, `blame`, `why` | the commands of the same names |
| `hypothesis_status` | `origins hypothesis status` |
| `schema` | `origins schema` |
| `propose` | `origins hypothesis propose --from` |

`index` gives every record with its name and its links to other records, without the
prose, so the agent can load the structure of a large tree and then read only the
records it needs. `schema` gives the JSON Schema of each kind of record, which the
[reference](../reference/schema) also lists.

## Proposals

`propose` takes a whole hypothesis file, frontmatter and body, and records it as an
open hypothesis in one commit. The agent sends a file such as this one:

```markdown
---
supports: [battesimi-borgo-san-lorenzo#1842-057]
edits:
  - op: set
    entity: person/rossi-giovanni
    path: name.given
    value: Giovanni Battista
---

# Was Giovanni baptised Giovanni Battista?
```

Origins pins the hypothesis to the last commit and adds `proposed-by` with the name
from `--as`. It refuses a file that names a proposer itself, and a proposal whose
changes can't be applied to the corpus. The commit carries the name too, so `log`,
`why` and the pages show who proposed the hypothesis:

```console
$ origins log -n 1
27f7c06c2a  2026-10-19  Propose giovanni-battista
            hypothesis.propose  hypothesis/giovanni-battista  (Rosa Bianchi)
            from battesimi-borgo-san-lorenzo#1842-057, by giovanni-battista, proposed by assistant
```

Then you test the hypothesis with `origins check --with`, and accept or reject it, as
[Hypotheses](./hypotheses) describes.

# Glossary

The words zk uses, the rules for keys, how a box is ordered, and what zk shows on a terminal. The fields themselves are in the README ("The box spec") and ADR 261115-0tp4.

## Words

| Term           | Also         | Meaning                                                                                                                                                                                                             |
| -------------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **slip box**   | box          | The whole collection of cards: a folder in an Obsidian vault with a `zk.toml` spec.                                                                                                                                 |
| **card**       | slip         | One note in the box: one thought, in plain words, linked to others with the reason stated.                                                                                                                          |
| **key**        | address      | A card's permanent address: its segments written together with no separator, such as `7a1`. It records where the card was placed and never changes.                                                                 |
| **segment**    |              | One run of digits or one run of letters in a key. They alternate, so a segment ends where digits meet letters.                                                                                                      |
| **level**      | depth        | A segment's position in the key, counting from 1. A card's level is the number of segments in its key. Odd levels are numbers, even levels are letters.                                                             |
| **sort key**   |              | A padded form of a key that sorts correctly as text. zk writes it in the card's `sort` field; it never appears in a name or a link.                                                                                 |
| **root**       |              | A card with a one-segment key (`7`), at level 1. It has no parent.                                                                                                                                                  |
| **parent**     |              | The card whose key is this key without its last segment: `7a` is the parent of `7a1`.                                                                                                                               |
| **child**      | comment      | A card whose parent is this card. A child elaborates on or answers its parent. The first child is `1` under a letter and `a` under a number.                                                                        |
| **sibling**    | continuation | A card with the same parent. The next sibling of `7a` is `7b`; this is how a thread continues.                                                                                                                      |
| **ancestor**   |              | The parent, its parent, and so on up to the root.                                                                                                                                                                   |
| **descendant** |              | A child, a child's child, and so on.                                                                                                                                                                                |
| **family**     | section      | A card and all its descendants. The tag search `tag:#zk/7` finds the family of card 7.                                                                                                                              |
| **kind**       |              | What sort of statement a card makes. The defaults are below; a box's spec can define its own.                                                                                                                       |
| **status**     |              | How far along a card is: `raw`, `proposed` or `linked`. It says nothing about whether the card is true.                                                                                                             |
| **source**     |              | A page for one thing read or heard: an article, a video, a book, a conversation. Cards that come from it link to it.                                                                                                |
| **provenance** |              | Where a source comes from: a URL, a link to another note in the vault, or a description ("a notebook from a class").                                                                                                |
| **harvest**    |              | Turning a source into cards: cut it into the pieces that carry a meaning, leave the filler on the source page, then place each piece in the tree and link it with a reason. A source is `extracted` once harvested. |
| **base**       |              | The Obsidian base (`<name> Base.base`) over the box's cards, with the views All, Raw and Proposed, sorted by the sort key. `zk init` writes it.                                                                     |
| **hub**        |              | The note beside the base that explains the box and embeds the base's views. Not the same as a hub card.                                                                                                             |
| **trail**      | walk         | An ordered sequence of cards chosen for one piece of writing. Reserved: zk has no trail commands yet. The word follows the trails of Vannevar Bush's 1945 memex.                                                    |

## Kinds

The seven kinds a box has when its spec defines none:

| Kind       | Holds                                                                                                                                                               | The check requires                        |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| `claim`    | What a source or person asserts, attributed in the text. It says nothing about whether it holds for anyone else.                                                    | A link to its source page.                |
| `fact`     | A statement held with strong confidence, inside a stated scope, with its evidence named. A claim becomes a fact only through a new card that supplies the evidence. | A link to its source page.                |
| `idea`     | The box owner's own thought.                                                                                                                                        | Nothing.                                  |
| `question` | An open question.                                                                                                                                                   | Nothing.                                  |
| `quote`    | Someone's exact words, with who said them and where, and only as much as the card needs.                                                                            | A link to its source page.                |
| `report`   | Something the owner did, saw or lived through: testimony, not proof. It says when and where it can.                                                                 | A link to its source page.                |
| `hub`      | An entrance to the graph for a person or a topic: a short description, then links, one sentence each.                                                               | At least one link under `## Connections`. |

## Statuses

| Status     | Meaning                                                                                                                         | What it needs                                                                         |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| `raw`      | Captured. It has a key, but no links yet, or its links are a guess. The box's to-do list.                                       | Nothing.                                                                              |
| `proposed` | Placed and linked: it sits at a sensible key and links to the cards it relates to, each link saying why. Waiting for the owner. | A link in or out. The check flags a card that is not `raw` and that nothing links to. |
| `linked`   | The owner has looked at the placement and the links and agrees. Only the owner sets it.                                         | The owner's word.                                                                     |

Sources have their own: `raw` (captured), `read` (summarised), `extracted` (cards made, listed in the source's `extracted` field).

## Keys

1. A key is one or more segments written together, with no separator and no padding.
2. Segments alternate: level 1 is a number, level 2 letters, level 3 a number, and so on. A number is a positive integer with no leading zero. Letters are lower case.
3. Letters count like spreadsheet columns: `a` to `z`, then `aa`, `ab` to `az`, then `ba` to `zz`, then `aaa`.
4. A new thread takes the next unused root number.
5. A card that continues X takes X's next unused sibling: `7a`, then `7b`.
6. A card that comments on X takes X's first free child: `7a`, then `7a1`.
7. A key is never changed or reused. A card that moves keeps its key; its links say where it belongs now.

zk takes one past the highest key it finds, so a gap left by a deleted card is never filled. It cannot see a deleted card, though: if the last of a set of siblings is deleted, its key can come back. Keep cards rather than delete them; a card that no longer fits can say so in its links.

`zk new` applies rules 4 to 7: `zk new idea "A thought"` takes the next root, and `--under KEY` the next child.

### One key, five forms

For the card said aloud as "7 a 1":

| Where                     | Written                                        | Set by             |
| ------------------------- | ---------------------------------------------- | ------------------ |
| The key field             | `"7a1"` (quoted, so Obsidian reads it as text) | `zk new`, `zk fix` |
| The file and the wikilink | `zk-7a1.md`, `[[zk-7a1]]`                      | `zk new`           |
| The tag                   | `zk/7/a/1`                                     | `zk new`, `zk fix` |
| The level field           | `3`                                            | `zk new`, `zk fix` |
| The sort field            | `"00000007/00000001/00000001"`                 | `zk new`, `zk fix` |

The key is the truth; the rest are derived from it, and `zk check` reports any that disagree. The prefix (`zk`) is the spec's `key_prefix`.

**Why a prefix.** A bare `7` could collide with any other note named `7`; `zk-7a1` is unique in a vault, and a search for it matches one thing. Obsidian tags nest by slash and match by segment, so `tag:#zk/7` finds card 7 and its descendants (`zk/7/a`, `zk/7/a/1`) but never `zk/70`. The prefix also stops a tag being all digits, which Obsidian rejects.

### The sort key

Each segment becomes a number, padded to eight digits, and the segments are joined with `/`. A number segment is its value; a letter segment is its spreadsheet index (`a` is 1, `z` is 26, `aa` is 27). So `7a1` is `00000007/00000001/00000001`, and `7aa` is `00000007/00000027`. Sorting on this one text field gives depth-first order. The padding is wide because it never appears in a name; if it ever has to widen, only the sort field is rewritten.

## Order

Two orders answer different questions. For the cards `1`, `1a`, `1a1`, `2`, `2a`, `2b`, `2b1`, `2b2`, `3`:

| Order                                     | Result                             | Answers                                                                                                            |
| ----------------------------------------- | ---------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| Depth first (the order of a physical box) | 1, 1a, 1a1, 2, 2a, 2b, 2b1, 2b2, 3 | "Follow this line of thought": a thread stays together, each card followed by its comments, then its next sibling. |
| Breadth first                             | 1, 2, 3, 1a, 2a, 2b, 1a1, 2b1, 2b2 | "What is in the box": the roots, then their children, and so on.                                                   |

Depth first is a sort on the sort field; `zk list` uses it, and `zk list --tree` draws it. Breadth first is a sort on the level field, then the sort field. Both work in an Obsidian base. A sort on one number field per level does not: Bases puts an empty value last, so a parent sorts after its children.

Anything that reads only file names cannot know the order. Numbers sort naturally in Obsidian's file list (`2`, `7`, `10`), but letters sort alphabetically, so the list is right until a letter level passes 26 siblings: `2aa` then lands before `2b`. A base is always right.

## A card's body

`zk new` writes the title as an `# H1`, an empty line for the text, and a `## Connections` heading.

- **The title** is a short label, a noun phrase, not a sentence. The H1 is the title; `aliases[0]` copies it as the card's nickname, which Obsidian's quick switcher searches. Edit the H1, then run `zk fix`, or use `zk retitle`.
- **The text** is the thought, in plain words. Who said it goes in the sentence ("The essay argues...").
- **Sources** are footnotes that link the source page: `[^1]: [[source-slug]]`.
- **Links to other cards** are wikilinks: `[[zk-7a1]]` shows the key, and a phrase from the sentence can be the link text: `[[zk-7a1|persistent agents]]`.
- **`## Connections`** is where each link says why the two cards belong together, one sentence per link.

## On a terminal

How zk looks when a person runs it on a terminal, and what everyone else gets.

| Term                 | Also | Meaning                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| -------------------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **terminal UI**      | TUI  | A full-screen view zk draws in the terminal, built with Bubble Tea. It opens only when stdin and stdout are both terminals and neither `--no-input` nor `ZK_NO_INPUT=1` is given, and only when the command's rule says so: `zk new` opens its terminal UI when the kind or the title is missing (ADR 261115-czb2), and `zk list` opens the interactive list whenever a terminal UI may open and `--json` is not given (ADR 261115-9pqw). |
| **interactive list** |      | The terminal UI of `zk list`: the cards on a full screen, as rows or as the tree, which you scroll, filter by typing after `/`, fold by branch in the tree, and open with enter. It prints nothing; quitting leaves the screen as it was (ADR 261115-9pqw).                                                                                                                                                                               |
| **rows**             |      | One of the two views of the interactive list: one line per card, as `zk list` prints them. `t` switches to the tree and back.                                                                                                                                                                                                                                                                                                             |
| **tree**             |      | The other view of the interactive list, and what `zk list --tree` prints: each card hangs under its parent with connector lines, or under its nearest ancestor shown when a filter hides the parent.                                                                                                                                                                                                                                      |
| **fold**             |      | In the tree of the interactive list, to hide the descendants of a card; the card stays, with the number hidden after it, as `[+3]`. Unfolding shows them again. Folds last until the list closes.                                                                                                                                                                                                                                         |
| **plain output**     |      | What a command prints when no terminal UI opens: lines of text, or one JSON object per line with `--json`. Agents and scripts always get plain output.                                                                                                                                                                                                                                                                                    |

## From Zettelkasten vocabulary

| Zettelkasten term                   | zk                                             |
| ----------------------------------- | ---------------------------------------------- |
| Zettelkasten                        | slip box                                       |
| Zettel                              | card                                           |
| Folgezettel, Luhmann ID             | key; a child or a sibling                      |
| literature note, bibliographic slip | source                                         |
| permanent note                      | idea (or a claim or fact, if attributed)       |
| fleeting note                       | a card with status `raw`                       |
| structure note                      | trail, or a hub card                           |
| Register                            | not used: search, tags and the base do the job |

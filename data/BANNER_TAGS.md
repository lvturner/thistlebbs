# Banner Color Tags

Tags are placed in `banner.txt` inside `{curly braces}`. They are
case-insensitive. Unknown tags are silently removed.

## Text Styles

| Tag         | Effect               | ANSI code |
|-------------|----------------------|-----------|
| `{reset}`   | Reset all attributes | `\x1b[0m` |
| `{bold}`    | Bold / bright        | `\x1b[1m` |
| `{dim}`     | Dim / dark           | `\x1b[2m` |
| `{italic}`  | Italic (not all terminals support this) | `\x1b[3m` |
| `{underline}` | Underline          | `\x1b[4m` |

## Foreground Colors

Standard (dark) colors:

| Tag         | Color  | ANSI code |
|-------------|--------|-----------|
| `{black}`   | Black  | `\x1b[30m` |
| `{red}`     | Red    | `\x1b[31m` |
| `{green}`   | Green  | `\x1b[32m` |
| `{yellow}`  | Yellow | `\x1b[33m` |
| `{blue}`    | Blue   | `\x1b[34m` |
| `{magenta}` | Magenta| `\x1b[35m` |
| `{cyan}`    | Cyan   | `\x1b[36m` |
| `{white}`   | White  | `\x1b[37m` |

Bright (light) colors:

| Tag              | Color      | ANSI code |
|------------------|------------|-----------|
| `{brightBlack}`  | Dark Gray  | `\x1b[90m` |
| `{brightRed}`    | Light Red  | `\x1b[91m` |
| `{brightGreen}`  | Light Green| `\x1b[92m` |
| `{brightYellow}` | Light Yellow| `\x1b[93m` |
| `{brightBlue}`   | Light Blue | `\x1b[94m` |
| `{brightMagenta}`| Light Magenta| `\x1b[95m` |
| `{brightCyan}`   | Light Cyan | `\x1b[96m` |
| `{brightWhite}`  | Bright White| `\x1b[97m` |

## Background Colors

Prefix with `bg:`. Same color names as foreground.

| Tag                   | Background      | ANSI code |
|-----------------------|-----------------|-----------|
| `{bg:black}`          | Black           | `\x1b[40m` |
| `{bg:red}`            | Red             | `\x1b[41m` |
| `{bg:green}`          | Green           | `\x1b[42m` |
| `{bg:yellow}`         | Yellow          | `\x1b[43m` |
| `{bg:blue}`           | Blue            | `\x1b[44m` |
| `{bg:magenta}`        | Magenta         | `\x1b[45m` |
| `{bg:cyan}`           | Cyan            | `\x1b[46m` |
| `{bg:white}`          | White           | `\x1b[47m` |
| `{bg:brightBlack}`    | Dark Gray       | `\x1b[100m` |
| `{bg:brightRed}`      | Light Red       | `\x1b[101m` |
| `{bg:brightGreen}`    | Light Green     | `\x1b[102m` |
| `{bg:brightYellow}`   | Light Yellow    | `\x1b[103m` |
| `{bg:brightBlue}`     | Light Blue      | `\x1b[104m` |
| `{bg:brightMagenta}`  | Light Magenta   | `\x1b[105m` |
| `{bg:brightCyan}`     | Light Cyan      | `\x1b[106m` |
| `{bg:brightWhite}`    | Bright White    | `\x1b[107m` |

## Example

```
{bold}{cyan}
   ████████░███████░██░░░░░░░██░░░░░░░░██░░░
   ██░░░░░░░██░░░░░░░██░░░░░░░██░░░░░░░░██░░░
   ████████░███████░░░████████░░░████████░░░░
   ██░░░░░░░██░░░░░░░██░░░░░░░██░░░░░░░░██░░░
   ██░░░░░░░██░░░░░░░██░░░░░░░██░░░░░░░░██░░░
   ████████░██░░░░░░░██░░░░░░░░░████████░░░░░
{reset}
{bold}{green} BBS {reset}

{yellow}░ Welcome to Thistle BBS ░{reset}
```

Edit `banner.txt` to customise the opening screen. The file is loaded
once at server startup.

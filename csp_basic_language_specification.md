# CSP-BASIC Language Specification & Reference Manual

**CSP-BASIC** (`cspbasic`) is a concurrent, structured dialect of 1984 IBM BASICA / GW-BASIC implemented in Go. It fuses classic line-numbered 8-bit/16-bit microcomputer BASIC—including `PEEK`/`POKE`, `DEF SEG`, non-echoing `INKEY$`, and CGA ANSI text-mode control—with **Tony Hoare's Communicating Sequential Processes (CSP)** (`SPAWN`, `CHAN`, `SEND`, `RECV`, `SELECT`) and modern functional extensions (thread-safe N-dimensional arrays, `@` record templates, recursive `DEF FN` stack frames, and first-class `FN APPLY` lambdas).

---

## 1. Program Structure & Execution Model

### 1.1 Line Numbers & Continuation-Based Program Counter (`pc`)
A CSP-BASIC program consists of numbered lines sorted in ascending order. Execution begins at the lowest line number and advances sequentially unless redirected by control-flow statements.

Internally, the virtual machine tracks execution using a two-part continuation program counter (`pc{idx, toks}`):
* **`idx`**: The index of the current line in the sorted program listing.
* **`toks`**: The remaining slice of tokens on the current line after a `:` statement separator.

Because control-flow frames (`FOR`, `WHILE`, `GOSUB`, `IF`, and `ON ERROR`) capture the exact intra-line continuation `pc`, multiple loops and subroutines can be packed onto a single line separated by colons (`:`).

### 1.2 Multi-Statement Lines (`:`) and Comments (`'` / `REM`)
* **Colon (`:`)**: Separates sequential statements on a single line, except after `IF ... THEN`, where the `IF` statement owns the remainder of the line (including any `ELSE` clause).
* **Comments**: Either the `REM` statement or a single apostrophe (`'`) begins a comment that extends to the end of the line. Numbered comment lines remain valid jump targets for `GOTO` and `GOSUB`.

```basic
10 ' Initialize configuration and run a single-line nested loop
20 N% = 3 : TOTAL% = 0
30 FOR I% = 1 TO N% : FOR J% = 1 TO N% : TOTAL% = TOTAL% + (I% * J%) : NEXT J% : NEXT I%
40 PRINT "Total = "; TOTAL% : END
```

---

## 2. Lexical Elements, Literals & Type Sigils

### 2.1 Numeric Literals
CSP-BASIC supports decimal integers, floating-point numbers, and 1984-style hexadecimal and octal literals:
* **Decimal**: `42`, `-15`, `3.14159`, `.5`
* **Hexadecimal (`&H` or `&h`)**: `&HB800`, `&HFF`, `&HFFFF`
* **Octal (`&O` or `&o`)**: `&O377`, `&O77`

### 2.2 String Literals & Escape Sequences
String literals are enclosed in double quotes (`"..."`). Both classic BASIC doubled quotes (`""`) and backslash escapes (`\"`, `\\`, `\n`, `\t`) are supported for embedding quotes inside string-backed lambdas:

```basic
10 PRINT "A hollow voice says ""Fool."""
20 RULE$ = "PRINT \"YOU SEE A \"; A$.NAME$"
```

### 2.3 Identifiers & Type Sigils
Identifiers are case-insensitive (`count%` and `COUNT%` refer to the same variable), begin with a letter, and may contain letters, digits, and underscores (`_`), followed by an optional **type sigil**:

| Sigil | Type | Default Value | Coercion & Semantics |
| :--- | :--- | :--- | :--- |
| **`$`** | String | `""` | Strictly typed as string; `+` always performs concatenation, even if contents are digits (`"123"`). |
| **`%`** | Integer | `"0"` | Truncates assigned numeric values to a 64-bit signed integer ($\text{int64}$). |
| **`@`** | Record | `""` | Composite structured record; packs and unpacks `.FIELD` members automatically. |
| **`!` / `#` / *(none)*** | Floating-Point | `"0"` | Double-precision floating-point ($\text{float64}$) formatted without trailing zeros. |

---

## 3. Variables, Scoping & Fiber Isolation

CSP-BASIC is concurrent by design. Every `SPAWN`ed worker runs in its own execution context (`execContext`) with isolated local variable storage so concurrent fibers never clobber each other's loop counters or scratch variables.

### 3.1 Scoping Rules
1. **Main Fiber (`isMain = true`)**: Variables read and write to the global `BasicVM` variable table by default, unless explicitly declared `LOCAL`.
2. **Spawned Worker Fibers (`isMain = false`)**:
   * **Reads**: Check the worker's private fiber-local map first; if unset, fall back to a thread-safe read (`RLock`) of the global variable table (Copy-on-Write semantics).
   * **Writes**: Write exclusively to the worker's private fiber-local map unless the variable was declared `SHARED`.
3. **Function Call Stack Frames (`DEF FN` & `FN APPLY`)**:
   * Every function or lambda invocation pushes a new `stackFrame`. Parameters and variables declared with `LOCAL` live strictly inside that call frame, enabling deep recursion without variable or `FOR`-loop collision.

### 3.2 Scope Declarations (`LOCAL`, `SHARED`, `LET`, `SWAP`)
* **`LOCAL v1, v2$, ...`**: Binds the listed variables to the current stack frame and initializes them to their sigil's default value.
* **`SHARED v1, v2$, ...`** *(alias `GLOBAL`)*: Binds the listed variables in a `SPAWN`ed worker directly to the mutex-protected global variable table.
* **`[LET] target = expr`**: Assigns `expr` to a scalar variable, array element, record field, or whole record.
* **`SWAP target1, target2`**: Atomically exchanges two scalars, array elements, record fields, or whole records.

```basic
10 COUNT% = 100
20 SPAWN 500
30 WAIT
40 PRINT "Global COUNT% = "; COUNT%; " | Shared STATUS$ = "; STATUS$
50 END

500 SHARED STATUS$
510 COUNT% = 999 : ' Modifies only the worker's private copy of COUNT%
520 STATUS$ = "WORKER_OK" : ' Modifies the global STATUS$ variable
530 RETURN
```

---

## 4. Operators & Expressions

Expressions are evaluated by a type-aware recursive-descent parser with the following precedence (from highest to lowest):

1. **Parentheses & Primary Access**: `( expr )`, `ARR(i)`, `REC@.FIELD`, `FN name(...)`, `FN APPLY(...)`, built-in functions.
2. **Unary Operators**: `-x` (negation), `+x` (unary plus), `#n` (channel identifier prefix).
3. **Exponentiation**: `x ^ y` (right-associative, $\text{math.Pow}(x, y)$).
4. **Multiplicative**:
   * `x * y` (multiplication)
   * `x / y` (floating-point division; division by $0$ returns `0`)
   * `x \ y` (integer division $\lfloor x / y \rfloor$)
   * `x MOD y` or `x % y` (integer remainder)
5. **Additive & Concatenation**:
   * `x + y`: Performs numeric addition if both operands are numeric; performs string concatenation if either operand has a `$` or `@` sigil or is a string literal.
   * `x - y`: Numeric subtraction.
6. **Relational Comparisons**: `=`, `<>`, `<`, `>`, `<=`, `>=`.
   * Compares numerically if both operands are non-string; compares lexicographically if either operand is a string.
   * Returns **`-1`** for **TRUE** and **`0`** for **FALSE** (authentic 1984 16-bit two's-complement boolean representation).
7. **Logical / Bitwise `NOT`**: `NOT x` (returns `0` if $x \ne 0$, or `-1` if $x = 0$).
8. **Logical / Bitwise `AND`**: 16-bit bitwise conjunction ($\text{int16}(x) \ \& \ \text{int16}(y)$).
9. **Logical / Bitwise `OR`, `XOR`, `EQV`, `IMP`**:
   * `OR`: $\text{int16}(x) \mid \text{int16}(y)$
   * `XOR`: $\text{int16}(x) \oplus \text{int16}(y)$
   * `EQV`: $\sim(\text{int16}(x) \oplus \text{int16}(y))$
   * `IMP`: $(\sim\text{int16}(x)) \mid \text{int16}(y)$

---

## 5. Control Flow & Error Trapping

### 5.1 Branching (`IF ... THEN ... ELSE`, `GOTO`, `GOSUB`, `ON`)
* **`IF cond THEN branch [ELSE branch]`**:
  * If `branch` is a bare line number (`IF X% = 1 THEN 500`), jumps directly to that line.
  * Otherwise, executes the inline statement chain (`IF X% = 1 THEN A% = 2 : PRINT A% ELSE A% = 0`).
* **`GOTO lineExpr`** / **`GOSUB lineExpr`** / **`RETURN [expr]`**:
  * `GOSUB` pushes the next statement's continuation `pc` onto the current frame's call stack.
  * `RETURN` pops the most recent `GOSUB` continuation. If the `GOSUB` stack is empty (such as at the end of a `SPAWN`ed worker or a multi-line `DEF FN`), `RETURN [expr]` cleanly terminates the fiber or returns `expr` from the function.
* **`ON expr GOTO line1, line2, ...`** / **`ON expr GOSUB line1, line2, ...`**:
  * Evaluates `expr` as a 1-based integer index ($1, 2, \dots, k$) and jumps or calls the corresponding target line. Out-of-range values fall through to the next statement.

### 5.2 Loops (`FOR ... NEXT` and `WHILE ... WEND`)
* **`FOR var = start TO end [STEP delta] ... NEXT var`**:
  * Supports positive and negative floating-point or integer steps, nested loops, and multi-statement single-line loops.
* **`WHILE cond ... WEND`**:
  * Evaluates `cond` before each iteration; if falsy (`0` or `""`), skips past the matching `WEND`.

### 5.3 Error Trapping (`ON ERROR GOTO`, `RESUME`, `ERR$`)
* **`ON ERROR GOTO lineExpr`**: Registers an error-trap handler for the current fiber (`ON ERROR GOTO 0` disables trapping).
* When a runtime or I/O error occurs, CSP-BASIC sets **`ERR$`** to the error message, records the faulting line and statement continuation, and jumps to the handler line.
* **`RESUME`**: Re-executes the faulting line.
* **`RESUME NEXT`**: Continues execution at the statement immediately following the faulting statement.
* **`RESUME lineExpr`**: Jumps to the specified line number.

```basic
10 ON ERROR GOTO 900
20 OPEN "nonexistent_file.dat" FOR INPUT AS #1
30 PRINT "Execution resumed safely after missing file!"
40 END

900 PRINT "[TRAPPED ERROR] "; ERR$
910 RESUME NEXT
```

---

## 6. Thread-Safe Arrays & Structured Records (`@`)

### 6.1 N-Dimensional Arrays (`DIM`, `REDIM`, `ERASE`, `UBOUND`)
Every array in CSP-BASIC is backed by a lock-free `ArrayMap` registry (`sync.Map`) and protected by its own per-array `sync.RWMutex`. Multiple goroutines can read and write to shared arrays concurrently without data races.

* **`DIM arr(d1 [, d2, ...])`**: Allocates an N-dimensional array with indices $0 \dots d_k$ inclusive per dimension. If an array is referenced without a prior `DIM`, it is atomically auto-dimensioned to $0 \dots 10$ per axis.
* **`REDIM arr(d1 [, d2, ...])`**: Reallocates and clears an existing array.
* **`ERASE arr1, arr2, ...`**: Deletes the specified arrays from memory.
* **`UBOUND(arr [, dim])`**: Returns the upper bound of dimension `dim` (1-based; defaults to `1`).

### 6.2 Record Templates (`DIM TEMPLATE` and `@` Sigil)
CSP-BASIC supports structured records via `DIM TEMPLATE` and the `@` sigil:

```basic
10 DIM TEMPLATE THING@(KEY$, NAME$, LOC%, WGT%, LIT%)
20 DIM ITEM@(10) AS THING@
30 DIM HERO@ AS THING@
```

* **Field Access**: Fields can be read or mutated using either array-of-structures syntax (`ITEM@(i).NAME$ = "Lantern"`) or structure-of-arrays syntax (`ITEM@.NAME$(i) = "Lantern"`), as well as scalar records (`HERO@.LOC% = 1`).
* **Whole-Record Operations**:
  * **`READ ITEM@(i)`**: Automatically reads one value from the `DATA` pool into each field defined in `THING@` in declaration order.
  * **Whole-Record Copy (`=`)**: `CHILD@ = ITEM@(i)` or `ITEM@(2) = ITEM@(1)` packs and copies all fields atomically.
  * **Whole-Record `SWAP`**: `SWAP ITEM@(1), ITEM@(2)` exchanges all fields between two records.
  * **Whole-Record `PRINT`**: `PRINT ITEM@(1)` formats all fields separated by tabs.

---

## 7. User-Defined Functions (`DEF FN`), Recursion & Lambdas (`FN APPLY`)

### 7.1 Single-Line and Multi-Line Recursive `DEF FN`
Functions can be defined as classic 1984 single-line formulas or as multi-line recursive blocks terminated by `END FN` (or `END DEF` / `ENDFN`). Both `FN name(...)` (with a space) and `FNname(...)` are accepted.

```basic
100 DEF FN FACTORIAL%(N%)
110   IF N% <= 1 THEN FN RETURN 1
120   FN RETURN N% * FN FACTORIAL%(N% - 1)
130 END FN

200 PRINT "5! = "; FN FACTORIAL%(5)
```

### 7.2 First-Class String Lambdas (`FN APPLY`)
Any string variable or expression can hold an anonymous formula or multi-statement BASIC routine and be executed via **`FN APPLY`**, **`FN APPLY%`**, **`FN APPLY$`**, or **`FN APPLY@`** (or bare `APPLY`):

* **Token Caching**: Lambda strings are tokenized once and cached in a lock-free `sync.Map` (`vm.lambdaCache`).
* **Automatic Positional Parameter Aliases**:
  * **1st Argument (`arg1`)**: Automatically bound to local `X`, `X%`, `X$`, `X@`, `A`, `A%`, `A$`, `A@`, `OBJ@`, `SELF@`, and `ARG1*`. If a record is passed, its fields are unpacked onto all aliases (so `A$.NAME$`, `A@.NAME$`, `X@.NAME$`, and `OBJ@.NAME$` all work automatically).
  * **2nd Argument (`arg2`)**: Bound to local `Y*`, `B*`, and `ARG2*`.
  * **3rd Argument (`arg3`)**: Bound to local `Z*`, `C*`, and `ARG3*`.
* **Explicit Arrow Header (`=>`)**: Lambdas may also declare explicit parameter names using `(P1, P2) => body`:

```basic
10 DIM TEMPLATE ITEM@(NAME$, OIL%)
20 DIM LANTERN@ AS ITEM@
30 LANTERN@.NAME$ = "BRASS LANTERN" : LANTERN@.OIL% = 5
40 ' Positional lambda (uses X%)
50 DECREMENT$ = "X% - 1"
60 ' Explicit parameter header lambda
70 CANSEE$ = "(R%, L%) => R% = 1 OR L% > 0"
80 ' Statement lambda accessing unpacked record fields via A$
90 IDENTIFY$ = "PRINT \"YOU SEE A \"; A$.NAME$; \" (OIL=\"; A$.OIL%; \")\""
100 LANTERN@.OIL% = FN APPLY%(DECREMENT$, LANTERN@.OIL%)
110 IF FN APPLY%(CANSEE$, 2, LANTERN@.OIL%) THEN FN APPLY(IDENTIFY$, LANTERN@)
```

---

## 8. CSP Concurrency & Channel Multiplexing

CSP-BASIC maps Go's goroutines and channels directly into 1984 BASIC syntax:

### 8.1 Concurrency Statements
* **`CHAN #n, bufSize`** *(or `CHAN name, bufSize`)*: Creates a channel with capacity `bufSize`. Using `bufSize = 0` creates an unbuffered synchronous rendezvous channel (ideal for lockstep turn coordination).
* **`SPAWN lineExpr`**: Launches a new concurrent goroutine fiber starting at `lineExpr`. The fiber runs until it hits `RETURN` (with an empty `GOSUB` stack) or `END`.
* **`SEND #n, exprList`**: Evaluates a `PRINT`-style expression list and sends the resulting string over channel `#n`.
* **`RECV #n, target`**: Receives a message from channel `#n` into `target` (scalar, array slot, or record) and sets `EOF$` to `"0"` (or `"1"` if the channel has been closed and drained).
* **`CLOSE [#n, #m, ...]`**: Safely closes the specified channels (or all open channels/files if called with no arguments). Buffered messages remain readable by `RECV` until drained.
* **`WAIT`**: Blocks until all `SPAWN`ed fibers have completed.
* **`SLEEP secondsExpr`**: Pauses the current fiber for `secondsExpr` seconds (supports fractional seconds such as `SLEEP 0.25`).

### 8.2 `SELECT ... CASE ... END SELECT` Multiplexing
`SELECT` multiplexes multiple channels and timeouts simultaneously using Go's runtime `reflect.Select`:

* **`CASE RECV #n, target [THEN stmt | line]`**: Fires when data is available on `#n`. Closed and drained channels are automatically evicted from the `SELECT` set; when all watched `RECV` channels reach EOF, `SELECT` sets `EOF$ = "1"` and exits past `END SELECT`.
* **`CASE SEND #n, exprList [THEN stmt | line]`**: Fires when channel `#n` can accept a send without blocking.
* **`CASE TIMEOUT secondsExpr [THEN stmt | line]`**: Fires if no channel case is ready within `secondsExpr` seconds.
* **`CASE ELSE`** *(or `CASE DEFAULT`)*: Executes immediately if no channel is ready (non-blocking poll).

```basic
10 CHAN #1, 4 : CHAN #2, 4
20 SPAWN 500 : SPAWN 600
30 SELECT
40   CASE RECV #1, FAST$ THEN PRINT "[#1] "; FAST$
50   CASE RECV #2, SLOW$
60     PRINT "[#2] "; SLOW$
70   CASE TIMEOUT 0.20 THEN PRINT "... waiting ..."
80 END SELECT
90 IF EOF$ = "0" THEN 30
100 WAIT : END

500 FOR I% = 1 TO 3 : SLEEP 0.10 : SEND #1, "Fast-" + STR$(I%) : NEXT I% : CLOSE #1 : RETURN
600 FOR I% = 1 TO 2 : SLEEP 0.30 : SEND #2, "Slow-" + STR$(I%) : NEXT I% : CLOSE #2 : RETURN
```

---

## 9. Unified Sequential File I/O & `DATA` Pool

### 9.1 Sequential Disk Files (`OPEN`, `PRINT #`, `WRITE #`, `INPUT #`, `LINE INPUT #`)
Disk files are backed by the exact same `BasicChan` descriptor engine as in-memory channels. Opening a file spawns a background reader or buffered writer goroutine connected to `#n`, meaning **`SELECT`, `RECV #n`, `SEND #n`, and `EOF(#n)` work on disk files identically to goroutine channels**.

* **`OPEN filename$ FOR INPUT | OUTPUT | APPEND AS #n`**
* **`PRINT #n, exprList`**: Writes formatted output to `#n` (with `\r\n` on file channels).
* **`WRITE #n, expr1, expr2, ...`**: Writes comma-separated values with strings automatically enclosed in double quotes (`"..."`).
* **`INPUT #n, target1, target2, ...`**: Reads and parses comma-separated fields from `#n`.
* **`LINE INPUT #n, var$`**: Reads an entire raw line from `#n` into `var$`.
* **`KILL filename$`**: Deletes a file from disk.
* **`NAME oldName$ AS newName$`**: Renames or moves a file.

### 9.2 Static `DATA` Stream (`DATA`, `READ`, `RESTORE`, `DREM`)
* **`DATA item1, item2, ...`**: Declares comma-separated constants pre-indexed at program load time.
* **`READ target1, target2, ...`**: Atomically reads the next items from the `DATA` pool into variables, array elements, or whole `@` records. Sets `DATA_EOF$ = "1"` if exhausted.
* **`RESTORE [lineExpr]`**: Resets the `DATA` cursor to the beginning of the program, or to the first `DATA` line at or after `lineExpr`.
* **`DREM()`**: Built-in function returning the number of unread `DATA` items remaining.

---

## 10. 1984 CRT Console & 1 MiB Real-Mode Memory Emulation

### 10.1 80x25 Text-Mode CRT & Keyboard (`CLS`, `LOCATE`, `COLOR`, `INKEY$`)
* **`CLS`**: Clears the terminal and resets the cursor to $(1, 1)$.
* **`LOCATE [row][, col]`**: Moves the cursor to 1-based `row` and `col`.
* **`COLOR fg[, bg]`**: Sets the 1984 CGA text color (`fg` $0 \dots 15$, `bg` $0 \dots 7$).
* **`BEEP`**: Emits the console bell (`\a`).
* **`INKEY$`**: Polls the hardware keyboard buffer without blocking or echoing. Returns `""` if no key is pressed, a 1-byte string for standard ASCII keys, or a 2-byte `CHR$(0) + CHR$(scanCode)` string for extended arrow/function keys.

### 10.2 1 MiB Segmented Memory (`DEF SEG`, `PEEK`, `POKE`)
CSP-BASIC emulates a thread-safe $1\text{ MiB}$ ($2^{20} = 1{,}048{,}576\text{ bytes}$) real-mode address space shared across all fibers:
$$\text{Physical Address} = (\text{Segment} \times 16 + \text{Offset}) \pmod{2^{20}}$$
* **`DEF SEG [= segmentExpr]`**: Sets the current 16-bit segment register (bare `DEF SEG` resets to `0`).
* **`POKE offsetExpr, byteExpr`**: Writes an 8-bit byte ($0 \dots 255$) to `offsetExpr`.
* **`PEEK(offsetExpr)`**: Reads the 8-bit byte at `offsetExpr`.

---

## 11. Built-In Functions Reference

| Function | Category | Description |
| :--- | :--- | :--- |
| `LEN(s$)` | String | Number of Unicode characters (runes) in `s$`. |
| `LEFT$(s$, n)` / `RIGHT$(s$, n)` | String | First or last `n` characters of `s$`. |
| `MID$(s$, start [, len])` | String | Substring starting at 1-based index `start`. |
| `INSTR([start,] hay$, needle$)` | String | 1-based index of `needle$` in `hay$` (`0` if not found). |
| `UCASE$(s$)` / `LCASE$(s$)` | String | Converts `s$` to uppercase or lowercase. |
| `TRIM$(s$)` | String | Strips leading and trailing whitespace from `s$`. |
| `CHR$(code)` / `ASC(s$)` | String | Converts between ASCII/Unicode code point and character. |
| `STR$(n)` / `VAL(s$)` | String | Converts a number to string or parses leading numeric value from `s$`. |
| `SPACE$(n)` / `SPC(n)` | String | Returns a string of `n` spaces. |
| `STRING$(n, ch$OrCode)` | String | Repeats character `ch$` (or ASCII code) `n` times. |
| `HEX$(n)` / `OCT$(n)` | String | Formats integer `n` in uppercase hexadecimal or octal. |
| `ABS(n)` / `SGN(n)` | Math | Absolute value of `n`; sign of `n` (`-1`, `0`, or `1`). |
| `INT(n)` / `FIX(n)` | Math | Floor ($\lfloor n \rfloor$) vs. truncation toward zero. |
| `SQR(n)` | Math | Square root $\sqrt{\max(0, n)}$. |
| `SIN(n)`, `COS(n)`, `TAN(n)`, `ATN(n)` | Math | Trigonometric functions (in radians). |
| `LOG(n)` / `EXP(n)` | Math | Natural logarithm $\ln(n)$ and exponential $e^n$. |
| `MIN(a, b)` / `MAX(a, b)` | Math | Minimum or maximum of two numbers. |
| `RND` / `RND()` | Math | Pseudo-random float in $[0.0, 1.0)$. |
| `RANDINT(min, max)` | Math | Pseudo-random integer in $[min, max]$ inclusive. |
| `UBOUND(arr [, dim])` | Array | Upper bound of array `arr` along 1-based dimension `dim`. |
| `DREM` / `DREM()` | Data | Count of unread items remaining in the `DATA` pool. |
| `QLEN(#n)` / `LOF(#n)` | CSP / I/O | Number of queued messages currently waiting in channel `#n`. |
| `EOF(#n)` / `EOF()` | CSP / I/O | Returns `1` if channel/file `#n` is closed and drained, else `0`. |
| `INKEY$` | Console | Non-blocking, non-echoing raw keystroke read. |
| `CSRLIN` / `POS(0)` | Console | Current 1-based cursor row (`CSRLIN`) and column (`POS`). |
| `TAB(col)` | Console | Spaces needed to advance cursor to 1-based column `col`. |
| `PEEK(offset)` | Memory | Reads byte at $(\text{DEF SEG} \times 16 + \text{offset}) \ \& \ \text{0xFFFFF}$. |
| `TIMER` / `TIMER()` | System | Elapsed seconds (with millisecond precision) since VM startup. |
# t-rex
Package t-rex is a Tiny Regular EXpressions implementation,
heavilly inspired by ["A Regular Expression Matcher"](http://www.cs.princeton.edu/courses/archive/spr09/cos333/beautiful.html)
and https://github.com/monolifed/tiny-regex-mod

### supports:
```
  '^'        Start anchor, matches start of string
  '$'        End anchor, matches end of string

  '*'        Asterisk, match zero or more (greedy, *? lazy)
  '+'        Plus, match one or more (greedy, +? lazy)
  '{m,n}'    Quantifier, match min. 'm' and max. 'n' (greedy, {m,n}? lazy)
  '{m}'                  exactly 'm'
  '{m,}'                 match min 'm' and max. MAX_QUANT
  '?'        Question, match zero or one (greedy, ?? lazy)

  '.'        Dot, matches any character except newline (\r, \n)
  '[abc]'    Character class, match if one of {'a', 'b', 'c'}
  '[^abc]'   Inverted class, match if NOT one of {'a', 'b', 'c'}
  '[a-zA-Z]' Character ranges, the character set of the ranges { a-z | A-Z }
  '\s'       Whitespace, \t \f \r \n \v and spaces
  '\S'       Non-whitespace
  '\w'       Alphanumeric, [a-zA-Z0-9_]
  '\W'       Non-alphanumeric
  '\d'       Digits, [0-9]
  '\D'       Non-digits
  '\X'       Character itself; X in [^sSwWdD] (e.g. '\\' is '\')
```
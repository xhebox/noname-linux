<!-- omit in toc -->
# Simple Starter Project
This starter project contains the scaffolding needed to integrate Clash with the Cabal and Stack build systems. It allows you to use dependencies from [Hackage](https://hackage.haskell.org/) easily.

<!-- omit in toc -->
# Table of Contents
- [Getting this project](#getting-this-project)
- [Building and testing this project](#building-and-testing-this-project)
  - [Stack (Windows, Linux, MacOS) [recommended]](#stack-windows-linux-macos-recommended)
  - [Cabal (Linux, MacOS)](#cabal-linux-macos)
  - [REPL](#repl)
  - [IDE support](#ide-support)
- [Project overview](#project-overview)
  - [my-clash-project.cabal](#namecabal)
  - [cabal.project](#cabalproject)
  - [stack.yaml](#stackyaml)
  - [src/](#src)
  - [tests/](#tests)
- [Change the license](#change-the-license)

# Getting this project
Stack users can run `stack new my-clash-project clash-lang/simple`. Cabal users can [download a zip](https://raw.githubusercontent.com/clash-lang/clash-starters/main/simple.zip) containing the project.

# Building and testing this project
There's a number of ways to build this project on your machine. The recommended way of doing so is using _Stack_, whose instructions will follow directly after this section.

## Stack (Windows, Linux, MacOS) [recommended]
Install Stack using your package manager or refer to the [How to install](https://docs.haskellstack.org/en/stable/README/#how-to-install) section of the [Stack manual](https://docs.haskellstack.org/en/stable/README/).

Build the project with:

```bash
stack build
```

To run the tests defined in `tests/`, use:

```bash
stack test
```

To compile the project to VHDL, run:

```bash
stack run clash -- Example.Project --vhdl
```

You can find the HDL files in `vhdl/`. The source can be found in `src/Example/Project.hs`.

## Cabal (Linux, MacOS)
**The following instructions only work for Cabal >=3.0 and GHC >=8.4.**

First, update your cabal package database:

```bash
cabal update
```

You only have to run the update command once. After that, you can keep rebuilding your project by running the build command:

```bash
cabal build
```

To run the tests defined in `tests/`, use:

```bash
cabal run test-library --enable-tests
cabal run doctests --enable-tests
```

To compile the project to VHDL, run:

```bash
cabal run clash -- Example.Project --vhdl
```

You can find the HDL files in `vhdl/`. The source can be found in `src/Example/Project.hs`.

## REPL
Clash offers a [REPL](https://en.wikipedia.org/wiki/Read%E2%80%93eval%E2%80%93print_loop) as a quick way to try things, similar to Python's `python` or Ruby's `irb`. Stack users can open the REPL by invoking:

```
stack run clashi
```

Cabal users use:

```
cabal run clashi
```

## IDE support
We currently recommend Visual Studio Code in combination with the _Haskell_ plugin. All you need to do is open this folder in VSCode; it will prompt you to install the plugin.

# Project overview
This section will give a tour of all the files present in this starter project. It's also a general introduction into Clash dependency management. It's not an introduction to Clash itself though. If you're looking for an introduction to Clash, read ["Clash.Tutorial" on Hackage](https://hackage.haskell.org/package/clash-prelude).

```
my-clash-project
├── bin
│   ├── Clash.hs
│   └── Clashi.hs
├── src
│   └── Example
│       └── Project.hs
├─── tests
│   ├── Tests
│   │   └── Example
│   │       └── Project.hs
│   ├── doctests.hs
│   └── unittests.hs
├── cabal.project
├── my-clash-project.cabal
└── stack.yaml
```

## my-clash-project.cabal
This is the most important file in your project. It describes how to build your project. Even though it ends in `.cabal`, Stack will use this file too. It starts of with meta-information:

```yaml
cabal-version:       2.4
name:                my-clash-project
version:             0.1
license:             BSD-2-Clause
author:              John Smith <john@example.com>
maintainer:          John Smith <john@example.com>
```

If you decide to publish your code on [Hackage](https://hackage.haskell.org/), this will show up on your package's front page. Take note of the license, it's set to `BSD-2-Clause` by default, but this might bee too liberal for your project. You can use any of the licenses on [spdx.org/licenses](https://spdx.org/licenses/). If none of those suit, remove the `license` line, add `license-file: LICENSE`, and add a `LICENSE` file of your choice to the root of this project. Moving on:

```yaml
common common-options
  default-extensions:
    BangPatterns
    BinaryLiterals
    ConstraintKinds
    [..]
    QuasiQuotes

    -- Prelude isn't imported by default as Clash offers Clash.Prelude
    NoImplicitPrelude
```

Clash's parent language is Haskell and its de-facto compiler, GHC, does a lot of heavy lifting before Clash gets to see anything. Because using Clash's Prelude requires a lot of extensions to be enabled to be used, we enable them here for all files in the project. Alternatively, you could add them where needed using `{-# LANGUAGE SomeExtension #-}` at the top of a `.hs` file instead. The next section, `ghc-options`, sets warning flags (`-Wall -Wcompat`) and flags that make GHC generate code Clash can handle.

Note that this whole section is a `common` "stanza". We'll use it as a template for any other definitions (more on those later). The last thing we add to the common section is some build dependencies:

```yaml
  build-depends:
    base,
    Cabal,

    -- clash-prelude will set suitable version bounds for the plugins
    clash-prelude >= 1.2.4 && < 1.6,
    ghc-typelits-natnormalise,
    ghc-typelits-extra,
    ghc-typelits-knownnat
```

These dependencies are fetched from [Hackage](https://hackage.haskell.org/), Haskell's store for packages. Next up is a `library` stanza. It defines where the source is located, in our case `src/`, and what modules can be found there. In our case that's just a single module, `Example.Project`.

```yaml
library
  import: common-options
  hs-source-dirs: src
  exposed-modules:
    Example.Project
  default-language: Haskell2010
```

Note that extra dependencies could be added by adding a `build-depends` line to this section. The following section defines a testsuite called _doctests_. Doctests are tests that are defined in the documentation of your project. We'll see this in action in [src/](#src).

```yaml
test-suite doctests
  import:           common-options
  type:             exitcode-stdio-1.0
  default-language: Haskell2010
  main-is:          doctests.hs
  hs-source-dirs:   tests

  build-depends:
    base,
    doctest >= 0.16.1 && < 0.18,
    clash-prelude
```

Last but not least, another testsuite stanza is defined:

```yaml
test-suite test-library
  import: common-options
  default-language: Haskell2010
  hs-source-dirs: tests
  type: exitcode-stdio-1.0
  ghc-options: -threaded
  main-is: unittests.hs
  other-modules:
    Tests.Example.Project
  build-depends:
    my-clash-project,
    QuickCheck,
    hedgehog,
    tasty >= 1.2 && < 1.3,
    tasty-hedgehog,
    tasty-th
```

These testsuites are executed when using `stack test` or `cabal test --enable-tests`. Note that Cabal swallows the output if more than one testsuite is defined, as is the case here. You might want to consider running the testsuites separately. More on tests in [/tests](#tests).

## cabal.project
A `cabal.project` file is used to configure details of the build, more info can be found in the [Cabal user documentation](https://cabal.readthedocs.io/en/latest/cabal-project.html). We use it to make Cabal always generate GHC environment files, which is a feature Clash needs when using Cabal. It also sets a flag for older versions of Clash, massively speeding up compilation. It is ignored by Stack.

```haskell
packages:
  my-clash-project.cabal

package clash-prelude
  -- 'large-tuples' generates tuple instances for various classes up to the
  -- GHC imposed maximum of 62 elements. This severely slows down compiling
  -- Clash, and triggers Template Haskell bugs on Windows. Hence, we disable
  -- it by default. This will be the default for Clash >=1.4.
  flags: -large-tuples

write-ghc-environment-files: always
```

`cabal.project` can be used to build multi-package projects, by extending `packages`.

## stack.yaml
While Cabal fetches packages straight from Hackage (with a bias towards the latest versions), Stack works through _snapshots_. Snapshots are an index of packages from Hackage know to work well with each other. In addition to that, they specify a GHC version. These snapshots are curated by the community and FP Complete and can be found on [stackage.org](https://www.stackage.org/).

```yaml
resolver: lts-17.13

extra-deps:
  # At the time of writing, no snapshot includes Clash 1.4 yet so we add it - and
  # its dependencies - manually.
  - lazysmallcheck-0.6
  - Stream-0.4.7.2
  - arrows-0.4.4.2
  - clash-prelude-1.4.2
  - clash-lib-1.4.2
  - clash-ghc-1.4.2
```

This project uses [lts-17.13](https://www.stackage.org/lts-17.13), which includes Clash 1.2.5. We've added the extra-deps section to make sure Stack fetches the latest version of Clash, 1.4.2, instead. The point of this exercise is to make reproducible builds. Or in other words, if a `stack build` works now, it will work in 10 years too.

Note: If you need a newer Clash version, simply change the version bounds in `my-clash-project.cabal` and follow the hints given by Stack.

## src/
This is where the source code of the project lives, as specified in `my-clash-project.cabal`. It contains a single file, `Example/Project.hs` which starts with:

```haskell
module Example.Project (topEntity, plus) where

import Clash.Prelude

-- | Add two numbers. Example:
--
-- >>> plus 3 5
-- 8
plus :: Signed 8 -> Signed 8 -> Signed 8
plus a b = a + b
```

`my-clash-project.cabal` enabled `NoImplicitPrelude` which enables the use of `Clash.Prelude` here. Next, a function `plus` is defined. It simply adds two numbers. Note that the example (`>>> plus 3 5`) gets executed by the _doctests_ defined for this project and checked for consistency with the result in the documentation (`8`).

```haskell
-- | 'topEntity' is Clash's equivalent of 'main' in other programming
-- languages. Clash will look for it when compiling 'Example.Project'
-- and translate it to HDL. While polymorphism can be used freely in
-- Clash projects, a 'topEntity' must be monomorphic and must use non-
-- recursive types. Or, to put it hand-wavily, a 'topEntity' must be
-- translatable to a static number of wires.
topEntity :: Signed 8 -> Signed 8 -> Signed 8
topEntity = plus
```

as the comment says `topEntity` will get compiled by Clash if we ask it to compile this module:

```
stack run clash -- Example.Project --vhdl
```

or

```
cabal run clash -- Example.Project --vhdl
```

We could instead ask it to synthesize `plus` instead:

```
stack run clash -- Example.Project --vhdl -main-is plus
```

If you want to play around with Clash, this is probably where you would put all the definitions mentioned in ["Clash.Tutorial" on Hackage](https://hackage.haskell.org/package/clash-prelude).

## tests/
Most of this directory is scaffolding, with the meat of the tests being defined in `tests/Tests/Example/Project.hs`. Writing good test cases is pretty hard: edge cases are easy to forget both in the implementation and tests. To this end, it's a good idea to use _fuzz testing_. In this project we use [Hedgehog](https://hedgehog.qa/):

```haskell
import Example.Project (plus)

prop_plusIsCommutative :: H.Property
prop_plusIsCommutative = H.property $ do
  a <- H.forAll (Gen.integral (Range.linear minBound maxBound))
  b <- H.forAll (Gen.integral (Range.linear minBound maxBound))
  plus a b === plus b a
```

This test generates two numbers `a` and `b` that fit neatly into domain of `Signed 8`, thanks to the use of `minBound` and `maxBound`. It then tests whether the `plus` operation commutes. I.e., whether `a + b ~ b + a`. All functions called `prop_*` are collected automatically:

```haskell
tests :: TestTree
tests = $(testGroupGenerator)
```

We can run the tests using `stack test` or `cabal run test-library --enable-tests`:

```
.
  Tests.Example.Project
    plusIsCommutative: OK
        ✓ plusIsCommutative passed 100 tests.

All 1 tests passed (0.00s)
```

# Change the license
By default `my-clash-project.cabal` sets its `license` field to `BSD-2-Clause`. You might want to change this.

import Prelude

import Test.Tasty

import qualified Tests.Example.Project

main :: IO ()
main = defaultMain $ testGroup "."
  [ Tests.Example.Project.tests
  ]

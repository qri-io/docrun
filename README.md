[![Qri](https://img.shields.io/badge/made%20by-qri-magenta.svg?style=flat-square)](https://qri.io)
[![License](https://img.shields.io/github/license/qri-io/startf.svg?style=flat-square)](./LICENSE)

# Docrun

`Docrun` is a utility for extracting code samples from documentation, running this code, and testing that it produces the expected result. The outcome is that sample code avoids going stale or drifting away from the actual implementation.

### Example usage:

Markdown source should look like this:

    <!-​-
    docrun:
      pass: true
    -​->
    `​``
    def func():
      return 1
    `​``

The `docrun` structure contains metadata on how to run the source code that follows it. In this case, `pass` being set to true specifies that the test automatically passes, which counts as a "trivial" success.

Further documentation on all of the options for the `docrun` structure will be added soon.
[![Qri](https://img.shields.io/badge/made%20by-qri-magenta.svg?style=flat-square)](https://qri.io)
[![License](https://img.shields.io/github/license/qri-io/startf.svg?style=flat-square)](./LICENSE)

# Docrun

`Docrun` is a utility for extracting code samples from documentation, running this code, and testing that it produces the expected result. The outcome is that sample code avoids going stale or drifting away from the actual implementation.

### Example usage

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

# Docrun structure

```
docrun:
  // Following four fields are mutually exclusive
  pass
  test
  command
  filltype
  // Optional fields
  lang
  save
```

## pass

Causes the test to trivially pass.

```
docrun:
  pass: true
```

## test

Run a sample of starlark code. Will execute the starlark code and check that it produces the expected output. Specially named variables like `ds` (dataset) and `ctx` (context) are automatically created ahead of time.

    <--
    docrun:
      test:
        setup:  ds.set_body(["a","b","c"])
        call:   transform(ds, ctx)
        actual: ds.get_body()
        expect: [["a","b","c","abc"]]
    -->
    ```
    def transform(ds, ctx):
      body = ds.get_body()
      accum = ""
      for entry in body:
        accum += entry
      body.append(accum)
      ds.set_body(body)
    ```
    
### setup

Called first to setup any necessary state before the main test execution.

### call

The entry point for this test case.

### actual

How to access the result of running this test case.

### expect

The expected result to compare against `actual`.

## command

Executes something on the command-line. Currently a stub, needs further implementation.

## filltype

Parses the example as a piece of structured data and uses qri/base/fill/struct to assign the result to an in-memory object. Checks that the example code is valid syntax and uses correct field names for the structured data.

    <!--
    docrun:
      filltype: dataset.Dataset
    -->
    ```yaml
    meta:
      title: Example dataset title
    ```
    
### filltype

The type of structured data. Can either be a general format specifier like "json" or "yaml", otherwise is the name of a structure known about by `docrun`.

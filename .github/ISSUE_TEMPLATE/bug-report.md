name: Bug Report
description: Report a bug encountered while operating `gomi`
labels: kind/bug
body:
  - type: textarea
    id: problem
    attributes:
      label: What happened?
      description: |
        Please provide as much info as possible. Not doing so may result in your bug not being addressed in a timely manner.
    validations:
      required: true

  - type: textarea
    id: expected
    attributes:
      label: What did you expect to happen?
    validations:
      required: true

  - type: textarea
    id: repro
    attributes:
      label: How can we reproduce it (as minimally and precisely as possible)?
    validations:
      required: true

  - type: textarea
    id: additional
    attributes:
      label: Anything else we need to know?

  - type: textarea
    id: version
    attributes:
      label: version
      value: |
        <details>

        ```console
        $ gomi --version
        # paste output here
        ```

        </details>
    validations:
      required: true

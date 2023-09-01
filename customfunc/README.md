# Custom Functions 

Based on HCL User Functions Extension. This is based on the HCL source at
https://github.com/hashicorp/hcl/tree/main/ext/userfunc

This HCL extension allows a calling application to support user-defined
custom functions.

Functions are defined via a specific block type, like this:

```hcl
custom_function "getCfnOutput" {
    description = "Get value of a Cloudformation Output"
    parameters = [
      {
          name = "region",
          description = "AWS region"
      },
      {
          name = "stack_name",
          description = "Cloudformation Stack Name"
      },
      {
          name = "output_name",
          description = "Name of Cloudformation Output"
      }
    ]
    command = <<-EOT
        aws --region ${region} cloudformation describe-stacks --stack-name ${stack_name} |
        jq -r '.Stacks[] | select(.StackName == "${stack_name}") | .Outputs[] | select(.OutputKey == "${output_name}") | .OutputValue'
    EOT
}
```

The extension is implemented as a pre-processor for `cty.Body` objects. Given
a body that may contain functions, the `DecodeCustomFunctions` function searches
for blocks that define functions and returns a functions map suitable for
inclusion in a `hcl.EvalContext`. It also returns a new `cty.Body` that
contains the remainder of the content from the given body, allowing for
further processing of remaining content.

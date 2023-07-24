// Based on userfunc extension in the HCL library,
// at https://github.com/hashicorp/hcl/tree/main/ext/userfunc
//
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package customfunc implements a HCL extension that allows user-defined
// custom functions in HCL configuration.
//
// Using this extension requires some integration effort on the part of the
// calling application, to pass any declared functions into a HCL evaluation
// context after processing.
//
// The function declaration syntax looks like this:
//
//			custom_function "greet" {
//			  params = [name]
//			  command = "echo \"Hello, ${name}!\""
//			}
//
//		 or using heredoc syntax:
//
//	        custom_function "greet" {
//			  params = [name]
//			  command = <<-EOT
//			    (echo "Hello, ${name}!"
//		         echo "Hello again, ${name}") |
//		        grep again
//		      EOT
//			}
//
// When a custom function is called, the string given for the "command"
// attribute is evaluated in an isolated evaluation context that defines variables
// named after the given parameter names.  The resulting command string is then
// executed in a shell.  The default shell used to execute the command is
// "/bin/sh" but you can use another shell by setting the environment variable
// SHELL.  For example, SHELL=/bin/bash
package customfunc

// aFunc
custom_function "aFunc" {
  params  = [a]
  command = "echo ${a}"
}

// cfnOutput is a func that...
custom_function "cfnOutput" {
  params  = [region, stack_name, output_name]
  command = <<-EOT
       ds="$(aws --region ${region} cloudformation describe-stacks --stack-name ${stack_name})"
       if [[ $? != 0 ]]; then
         echo "Could not describe stack ${stack_name}: $ds"
         exit 1
       fi
       echo "$${ds}" |
       jq -re '.Stacks[] |
               select(.StackName == "${stack_name}") |
               .Outputs[] |
               select(.OutputKey == "${output_name}") |
               .OutputValue' ||
       (echo "Could not find CFN output ${output_name} in stack ${stack_name}"
        exit 1)
  EOT
}

// dev context
public_key "DEV20230604" {
  base64 = <<-EOT
      eyJwcmltYXJ5S2V5SWQiOjYxMzg0NDUsImtleSI6W3sia2V5R
      GF0YSI6eyJ0eXBlVXJsIjoidHlwZS5nb29nbGVhcGlzLmNvbS
      9nb29nbGUuY3J5cHRvLnRpbmsuSHBrZVB1YmxpY0tleSIsInZ
      hbHVlIjoiRWdZSUFSQUJHQUlhSUpPOWpZOXVhVkljd2NsNEV1
      S1lqUmFDSklzVktkR29DNmFrZFlUUDFWeEsiLCJrZXlNYXRlc
      mlhbFR5cGUiOiJBU1lNTUVUUklDX1BVQkxJQyJ9LCJzdGF0dX
      MiOiJFTkFCTEVEIiwia2V5SWQiOjYxMzg0NDUsIm91dHB1dFB
      yZWZpeFR5cGUiOiJUSU5LIn1dfQo=
  EOT
}

public_key "TIM" {
  base64 = <<-EOT
    eyJwcmltYXJ5S2V5SWQiOjYxMzg0NDUsImtleSI6W3sia2V5RGF0YSI6eyJ0eXBl
    VXJsIjoidHlwZS5nb29nbGVhcGlzLmNvbS9nb29nbGUuY3J5cHRvLnRpbmsuSHBr
    ZVB1YmxpY0tleSIsInZhbHVlIjoiRWdZSUFSQUJHQUlhSUpPOWpZOXVhVkljd2Ns
    NEV1S1lqUmFDSklzVktkR29DNmFrZFlUUDFWeEsiLCJrZXlNYXRlcmlhbFR5cGUi
    OiJBU1lNTUVUUklDX1BVQkxJQyJ9LCJzdGF0dXMiOiJFTkFCTEVEIiwia2V5SWQi
    OjYxMzg0NDUsIm91dHB1dFByZWZpeFR5cGUiOiJUSU5LIn1dfQo=
  EOT
}

// TIMMY
public_key "TIMMY" {
}

public_key "TIMMY2" {
}

public_key "TIMMY3" {
}

public_key "TIMMY9" {
  base64 = <<-EOT
    eyJwcmltYXJ5S2V5SWQiOjYxMzg0NDUsImtleSI6W3sia2V5RGF0YSI6eyJ0eXBl
    VXJsIjoidHlwZS5nb29nbGVhcGlzLmNvbS9nb29nbGUuY3J5cHRvLnRpbmsuSHBr
    ZVB1YmxpY0tleSIsInZhbHVlIjoiRWdZSUFSQUJHQUlhSUpPOWpZOXVhVkljd2Ns
    NEV1S1lqUmFDSklzVktkR29DNmFrZFlUUDFWeEsiLCJrZXlNYXRlcmlhbFR5cGUi
    OiJBU1lNTUVUUklDX1BVQkxJQyJ9LCJzdGF0dXMiOiJFTkFCTEVEIiwia2V5SWQi
    OjYxMzg0NDUsIm91dHB1dFByZWZpeFR5cGUiOiJUSU5LIn1dfQo=
  EOT
}

context "dev" {
  //enforce_schemas = ["zoom"]
  /*vars = {
      legacy_stack_name  = "dev"
      container_env_name = "dev"
      region = "us-west-2"
    }*/
  exports = {
    #REGION      = "${self.vars.region}"
    DB_USERNAME = "db"
    DB_PASSWORD = decrypt("dev20230604", "c3VwZXJTZWNyZXRQcm9k")
    DB_PORT     = "83306"
  }
}

context "production" {
  exports = {
    DB_USERNAME = "db"
    #DB_PASSWORD = decrypt("dev20230604", "c3VwZXJTZWNyZXRQcm9k")
    DB_PORT = "3306"
  }
}

context "staging" {
  #    vars = {
  #      legacy_stack_name  = "Staging"
  #      container_env_name = "staging"
  #      region             = "us-west-2"
  #    }
  exports = {
    //DB_PASSWORD = decrypt("DEV20230604", "AQBdqk2vlNXrUBtzG4j+GMBfHsemAKTMkvB/wYPizMTp25pvFm+MO2B3fDipNpCbJOG8kx2A9Lr+n4n9D80Y6J/7+7Vhaw/NN5A=")
    //REGION = foo.baz
    DB_PORT     = cfnOutput("us-west-2", "flex3", "Bastion1IPAddr")
    DB_USERNAME = "db"
  }
}


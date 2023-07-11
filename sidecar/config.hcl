custom_function "cfnOutput" {
    command = <<-EOT
        aws --region $1 cloudformation describe-stacks --stack-name $2 | 
        jq -r '.Stacks[] | select(.StackName == "$2") | .Outputs[] | select(.OutputKey == "$3") | .OutputValue'
    EOT
      
}

public_key "dev20230604" {
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

environment "dev" {
    region = "us-west-2"
    exports {
      REGION      = "$region"
      DB_USERNAME = "db"
      DB_PASSWORD = "db"
      DB_PORT     = "83306"
    }
}

environment "staging" {
    region = "us-west-2"
    exports {
      DB_USERNAME = "db"
      DB_PASSWORD = decrypt("dev20230604", 'AQBdqk2Z9iY1km3gT8zffMANtgOy12liTUyMrTdXZj1CCN3nZ8bdgs0hKLnA8sEI/fhyXldQjksanQZZ')
      DB_PORT     = cfnOutput($region, "staging", "db_port")   # can get reference $environment instead of hard coding?
    }
}

environment "production" {
    region = "us-west-2"
    exports {
        DB_USERNAME = "db"
        DB_PASSWORD = decrypt("dev20230604", "c3VwZXJTZWNyZXRQcm9k")
        DB_PORT = "3306"
    }
}

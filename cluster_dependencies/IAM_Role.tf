#=====================================================================================================================================
# The user and access key
#=====================================================================================================================================

resource "aws_iam_user" "Cluster_deployer" {
  name = "Cluster_deployer"
}

resource "aws_iam_access_key" "Cluster_deployer_key" {
  user    = aws_iam_user.Cluster_deployer.name
}

#=====================================================================================================================================
# Some outputs to be used from code outside the module
#=====================================================================================================================================

output IAM_User_name {
  value = aws_iam_user.Cluster_deployer.name
}

output IAM_User_Access_Key_id {
  value = aws_iam_access_key.Cluster_deployer_key.id
}

output IAM_User_Access_key_Secret {
  value = aws_iam_access_key.Cluster_deployer_key.secret
}
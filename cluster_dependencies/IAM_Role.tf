/*provider "aws" {
  region = "eu-west-1"
}*/

#=====================================================================================================================================
# The IAM Role that will be used to deploy the cluster. Will be used from Openshift-Installer
#=====================================================================================================================================

# resource "aws_iam_role" "Disconnected_cluster_deployer" {
#   name = "Disconnected_cluster_deployer"

#   assume_role_policy = jsonencode({
#     Version = "2012-10-17",
#     Statement = [
#       {
#         Action = "sts:AssumeRole",
#         Principal = {
#           Service = "ec2.amazonaws.com"
#         },
#         Effect = "Allow",
#         Sid = ""
#       }
#     ]
#   })
# }
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
# All data source resources for the policies
#=====================================================================================================================================

# resource "aws_iam_policy_attachment" "AllPermissions_attachment" {
#   name       = "AllPermissionsAttachment"
#   roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
#   policy_arn = aws_iam_policy.Installer_policy.arnIAM_User_Access_key
# }

output IAM_User_name {
  value = aws_iam_user.Cluster_deployer.name
}

output IAM_User_Access_Key_id {
  value = aws_iam_access_key.Cluster_deployer_key.id
}

output IAM_User_Access_key_Secret {
  value = aws_iam_access_key.Cluster_deployer_key.secret
}
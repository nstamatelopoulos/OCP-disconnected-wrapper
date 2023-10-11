/*provider "aws" {
  region = "eu-west-1"
}*/

#=====================================================================================================================================
# The IAM Role that will be used to deploy the cluster. Will be used from Openshift-Installer
#=====================================================================================================================================

resource "aws_iam_role" "Disconnected_cluster_deployer" {
  name = "Disconnected_cluster_deployer"

  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Action = "sts:AssumeRole",
        Principal = {
          Service = "ec2.amazonaws.com"
        },
        Effect = "Allow",
        Sid = ""
      }
    ]
  })
}
#=====================================================================================================================================
#
#=====================================================================================================================================

#=====================================================================================================================================
# All data source resources for the policies
#=====================================================================================================================================

resource "aws_iam_policy_attachment" "EC2Permissions_attachment" {
  name       = "EC2PermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.EC2_policy.arn
}

resource "aws_iam_policy_attachment" "VPCPermissions_attachment" {
  name       = "VPCPermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.Vpc_policy.arn
}

resource "aws_iam_policy_attachment" "LBPermissions_attachment" {
  name       = "LBPermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.LB_policy.arn
}

resource "aws_iam_policy_attachment" "LBv2Permissions_attachment" {
  name       = "LBv2PermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.LBv2_policy.arn
}

resource "aws_iam_policy_attachment" "IAMPermissions_attachment" {
  name       = "IAMPermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.Iam_resources_policy.arn
}

resource "aws_iam_policy_attachment" "Route53Permissions_attachment" {
  name       = "Route53PermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.Route53_policy.arn
}

resource "aws_iam_policy_attachment" "S3Permissions_attachment" {
  name       = "S3PermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.S3_policy.arn
}

resource "aws_iam_policy_attachment" "S3operatorsPermissions_attachment" {
  name       = "S3operatorsPermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.S3_operators_policy.arn
}

resource "aws_iam_policy_attachment" "DeletingClusterPermissions_attachment" {
  name       = "DeletingClusterPermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.Deleting_cluster_resources.arn
}

resource "aws_iam_policy_attachment" "DeletingNetworkPermissions_attachment" {
  name       = "DeletingNetworkPermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.Deleting_network_resources.arn
}

resource "aws_iam_policy_attachment" "UntagRolePermissions_attachment" {
  name       = "UntagRolePermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.Untag_Role.arn
}

resource "aws_iam_policy_attachment" "CreateManifestsPermissions_attachment" {
  name       = "CreateManifestsPermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.Create_manifests_policy.arn
}

resource "aws_iam_policy_attachment" "QuotasPolicyPermissions_attachment" {
  name       = "QuotasPolicyPermissionsAttachment"
  roles      = [aws_iam_role.Disconnected_cluster_deployer.name]
  policy_arn = aws_iam_policy.Quotas_policy.arn
}

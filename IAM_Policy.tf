#=====================================================================================================================================
# All policies required for the IAM role to deploy a cluster on AWS
#=====================================================================================================================================

resource "aws_iam_user_policy" "Installer_policy_document" {
  name        = "Installer_policy_document"
  user        = aws_iam_user.Cluster_deployer.name
  policy      = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
      "ec2:*",
      "elasticloadbalancing:*",
      "iam:AddRoleToInstanceProfile",
      "iam:CreateInstanceProfile",
      "iam:CreateRole",
      "iam:DeleteInstanceProfile",
      "iam:DeleteRole",
      "iam:DeleteRolePolicy",
      "iam:GetInstanceProfile",
      "iam:GetRole",
      "iam:GetRolePolicy",
      "iam:GetUser",
      "iam:ListInstanceProfilesForRole",
      "iam:ListRoles",
      "iam:ListUsers",
      "iam:PassRole",
      "iam:PutRolePolicy",
      "iam:RemoveRoleFromInstanceProfile",
      "iam:SimulatePrincipalPolicy",
      "iam:TagRole",
      "iam:CreateServiceLinkedRole",
      "iam:TagInstanceProfile",
      "route53:ChangeResourceRecordSets",
      "route53:ChangeTagsForResource",
      "route53:CreateHostedZone",
      "route53:DeleteHostedZone",
      "route53:GetChange",
      "route53:GetHostedZone",
      "route53:ListHostedZones",
      "route53:ListHostedZonesByName",
      "route53:ListResourceRecordSets",
      "route53:ListTagsForResource",
      "route53:UpdateHostedZoneComment",
      "s3:*",
      "autoscaling:DescribeAutoScalingGroups",
      "elasticloadbalancing:DeleteTargetGroup",
      "elasticloadbalancing:DescribeTargetGroups",
      "iam:DeleteAccessKey",
      "iam:DeleteUser",
      "iam:ListAttachedRolePolicies",
      "iam:ListInstanceProfiles",
      "iam:ListRolePolicies",
      "iam:ListUserPolicies",
      "tag:GetResources",
      "iam:UntagRole",
      "iam:DeleteAccessKey",
      "iam:DeleteUser",
      "iam:DeleteUserPolicy",
      "iam:GetUserPolicy",
      "iam:ListAccessKeys",
      "iam:PutUserPolicy",
      "iam:TagUser",
      "servicequotas:ListAWSDefaultServiceQuotas",
      "tag:UntagResources"
      ],

      "Resource": "*"
    }
  ]
}
EOF
}

output IAM_User_Policy_Name {
  value = aws_iam_user_policy.Installer_policy_document.name
}
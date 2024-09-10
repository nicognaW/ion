import { apigateway, iam, getRegionOutput } from "@pulumi/aws";

export function setupApiGatewayAccount(namePrefix: string) {
  const account = apigateway.Account.get(
    `${namePrefix}APIGatewayAccount`,
    "APIGatewayAccount",
  );

  return account.cloudwatchRoleArn.apply((arn) => {
    if (arn) return account;

    const region = getRegionOutput(undefined, { parent }).name;
    const isAWSCN = region.apply((region) => region.startsWith("cn-"));
    if (isAWSCN) {
      console.log("AWS China detected.");
    }

    const role = new iam.Role(
      `APIGatewayPushToCloudWatchLogsRole`,
      {
        assumeRolePolicy: JSON.stringify({
          Version: "2012-10-17",
          Statement: [
            {
              Effect: "Allow",
              Principal: {
                Service: "apigateway.amazonaws.com",
              },
              Action: "sts:AssumeRole",
            },
          ],
        }),
        managedPolicyArns: [
          isAWSCN ? "arn:aws-cn:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs" : "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs",
        ],
      },
      { retainOnDelete: true },
    );

    return new apigateway.Account(`${namePrefix}APIGatewayAccountSetup`, {
      cloudwatchRoleArn: role.arn,
    });
  });
}

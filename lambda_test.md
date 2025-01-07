## instructions
aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin 323258005321.dkr.ecr.us-west-2.amazonaws.com
docker build -f Dockerfile.lambda -t connector-test .
docker tag connector-test:latest 323258005321.dkr.ecr.us-west-2.amazonaws.com/connector-test:latest
docker push 323258005321.dkr.ecr.us-west-2.amazonaws.com/connector-test:latest
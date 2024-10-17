import grpc from "k6/net/grpc";
import { check, sleep } from "k6";

const client = new grpc.Client();
client.load(["calculator"], "./calculator.proto");

export const options = {
  vus: 100,
  duration: "30s",
};

export default () => {
  performOperation("Add");
  performOperation("Subtract");
  performOperation("Multiply");
  performOperation("Divide");
  sleep(1);
};

function performOperation(method) {
  client.connect("localhost:6000", {
    plaintext: true,
  });

  const data = { a: 18, b: 3 };
  const func = "calculator.Calculator/" + method;
  const response = client.invoke(func, data);

  check(response, {
    "status is OK": (r) => r && r.status === grpc.StatusOK,
  });

  client.close();
}

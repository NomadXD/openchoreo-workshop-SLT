export interface RuntimeConfig {
  chatGatewayHttpUrl: string;
  chatGatewayWsUrl: string;
  subscriptionServiceUrl: string;
  networkOpsServiceUrl: string;
}

declare global {
  interface Window {
    __CONFIG__?: Partial<RuntimeConfig>;
  }
}

const DEFAULTS: RuntimeConfig = {
  chatGatewayHttpUrl: 'http://localhost:8080',
  chatGatewayWsUrl: 'ws://localhost:8080',
  subscriptionServiceUrl: 'http://localhost:8081',
  networkOpsServiceUrl: 'http://localhost:8082',
};

const runtimeConfig = typeof window !== 'undefined' ? window.__CONFIG__ : undefined;

export const config: RuntimeConfig = {
  chatGatewayHttpUrl: runtimeConfig?.chatGatewayHttpUrl ?? DEFAULTS.chatGatewayHttpUrl,
  chatGatewayWsUrl: runtimeConfig?.chatGatewayWsUrl ?? DEFAULTS.chatGatewayWsUrl,
  subscriptionServiceUrl: runtimeConfig?.subscriptionServiceUrl ?? DEFAULTS.subscriptionServiceUrl,
  networkOpsServiceUrl: runtimeConfig?.networkOpsServiceUrl ?? DEFAULTS.networkOpsServiceUrl,
};

export type AppSettings = {
  name: string;
  env: string;
  language: string;
  secret: string;
};

export type LogSettings = {
  level: string;
  channel: string;
  path: string;
  format: string;
  maxSizeMb: number;
  maxAgeDays: number;
  compress: boolean;
};

export type ServerSettings = {
  host: string;
  port: number;
  trustedProxies: string;
  corsAllowedOrigins: string[];
  corsAllowCredentials: boolean;
  sslEnabled: boolean;
  sslCertFile: string;
  sslKeyFile: string;
};

export type S3Settings = {
  endpoint: string;
  region: string;
  bucket: string;
  accessKey: string;
  secretKey: string;
  usePathStyle: boolean;
};

export type StorageSettings = {
  driver: 'local' | 's3';
  path: string;
  s3: S3Settings;
};

export type ThemeSettings = {
  primary: string;
  secondary: string;
  success: string;
  info: string;
  warning: string;
  error: string;
  neutral: string;
};

export type TimingSettings = {
  serialPort: string;
  baudRate: number;
  poolLength: number;
  simulateEvent: number;
  replayPath: string;
  recording: boolean;
};

export type WebhookSettings = {
  // Comma-separated CIDRs or IPs of private or local networks webhooks may be delivered to
  allowedNetworks: string;
};

/** The config file a save writes to, with an empty path when the save creates a new one. */
export type ConfigFile = {
  path: string;
  exists: boolean;
};

/** Where the value of a setting comes from, with the name of the flag, the variable, or the file. */
export type ConfigSource = {
  kind: 'default' | 'file' | 'env' | 'flag';
  name: string;
};

/** Holds the sections that ConfigResponse and ConfigSaveRequest share. */
export type ConfigSettings = {
  app: AppSettings;
  log: LogSettings;
  server: ServerSettings;
  storage: StorageSettings;
  theme: ThemeSettings;
  timing: TimingSettings;
  webhook: WebhookSettings;
};

/**
 * The configuration to save. A secret sent empty keeps its saved value,
 * unless its path is listed in clearSecrets.
 */
export type ConfigSaveRequest = ConfigSettings & {
  clearSecrets: string[];
};

/** The configuration without the values of its secrets, with the defaults and sources by path. */
export type ConfigResponse = ConfigSettings & {
  secrets: Record<string, boolean>;
  defaults: ConfigSettings;
  sources: Record<string, ConfigSource>;
  file: ConfigFile;
};

export type ConfigTestResponse = {
  ok: boolean;
  message?: string;
};

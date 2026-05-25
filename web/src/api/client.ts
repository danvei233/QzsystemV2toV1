import axios from "axios";

const http = axios.create({ baseURL: "/admin/api", timeout: 20000 });

http.interceptors.request.use((config) => {
  const token = localStorage.getItem("xhp_token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem("xhp_token");
      window.dispatchEvent(new Event("xhp:logout"));
    }
    return Promise.reject(error);
  }
);

export interface RequestLog {
  id: number;
  trace_id: string;
  client_ip: string;
  downstream_method: string;
  downstream_path: string;
  downstream_headers: string;
  downstream_body: string;
  upstream_method: string;
  upstream_url: string;
  upstream_headers: string;
  upstream_body: string;
  upstream_response_status?: number;
  upstream_response_headers?: string;
  upstream_response_body?: string;
  response_status: number;
  response_headers: string;
  response_body: string;
  success: boolean;
  cache_hit: boolean;
  message: string;
  duration_ms: number;
  created_at: string;
}

export interface LogQuery {
  keyword?: string;
  path?: string;
  success?: string;
  start_at?: string;
  end_at?: string;
  limit?: number;
  offset?: number;
}

export interface EndpointErrorStat {
  path: string;
  total: number;
  success: number;
  failed: number;
  error_rate: number;
  avg_duration_ms: number;
  last_message: string;
  last_seen_at: string;
}

export interface AdminSettings {
  server_addr: string;
  admin_username: string;
  upstream_base_url: string;
  upstream_api_key_set: boolean;
  timeout: string;
  cache_ttl: string;
  max_body_bytes: number;
  require_api_key: boolean;
  accepted_api_keys: string[];
  database_dsn: string;
  log_retention_days: number;
  log_max_size_mb: number;
}

export interface UpdateSettingsPayload {
  admin_password?: string;
  upstream_base_url: string;
  upstream_api_key?: string;
  timeout: string;
  cache_ttl: string;
  max_body_bytes: number;
  require_api_key: boolean;
  accepted_api_keys: string[];
  log_retention_days: number;
  log_max_size_mb: number;
}

export const api = {
  login: (payload: { username: string; password: string }) => http.post("/login", payload),
  logout: () => http.post("/logout"),
  me: () => http.get("/me"),
  logs: (params: LogQuery) => http.get<{ items: RequestLog[]; total: number }>("/logs", { params }),
  log: (id: number) => http.get<RequestLog>(`/logs/${id}`),
  errorStats: (params: LogQuery) => http.get<{ items: EndpointErrorStat[] }>("/stats/error-rates", { params }),
  settings: () => http.get<AdminSettings>("/settings"),
  updateSettings: (payload: UpdateSettingsPayload) => http.patch("/settings", payload)
};

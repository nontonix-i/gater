import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card"
import { Badge } from "../components/ui/badge"
import { Separator } from "../components/ui/separator"

function Code({ children }: { children: React.ReactNode }) {
  return <code className="text-sm bg-muted px-1.5 py-0.5 rounded font-mono">{children}</code>
}

function Pre({ children }: { children: React.ReactNode }) {
  return <pre className="text-sm bg-muted p-3 rounded-lg overflow-x-auto font-mono">{children}</pre>
}

function Endpoint({ method, path, children }: { method: string; path: string; children: React.ReactNode }) {
  const colorMap: Record<string, string> = {
    GET: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
    POST: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
    PUT: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
    DELETE: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  }
  return (
    <div className="border rounded-lg p-4 space-y-2">
      <div className="flex items-center gap-2">
        <span className={`text-xs font-bold px-2 py-0.5 rounded ${colorMap[method] || ""}`}>{method}</span>
        <Code>{path}</Code>
      </div>
      {children}
    </div>
  )
}

export default function DocsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">API Documentation</h1>
        <p className="text-muted-foreground">REST API reference for Gater</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Base URL</CardTitle>
        </CardHeader>
        <CardContent>
          <Pre>{`http://your-server:8080/api/v1`}</Pre>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Authentication</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Most endpoints require authentication via API key. Provide it in one of these ways:
          </p>
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Badge variant="outline">Header</Badge>
              <Code>X-API-Key: your-api-key</Code>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="outline">Query</Badge>
              <Code>?api_key=your-api-key</Code>
            </div>
            <p className="text-xs text-muted-foreground">
              Get your API key from Settings page or via <Code>/auth/login</Code>.
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Auth</CardTitle>
          <p className="text-sm text-muted-foreground">Public endpoints (no auth required)</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <Endpoint method="POST" path="/auth/register">
            <p className="text-sm text-muted-foreground">Register a new user account.</p>
            <p className="text-xs font-medium mt-1">Request Body:</p>
            <Pre>{JSON.stringify({ email: "user@example.com", password: "secret", name: "User" }, null, 2)}</Pre>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ token: "gater-key-...", user: { id: "...", email: "user@example.com", name: "User", api_key: "gater-key-..." } }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="POST" path="/auth/login">
            <p className="text-sm text-muted-foreground">Login with email & password.</p>
            <p className="text-xs font-medium mt-1">Request Body:</p>
            <Pre>{JSON.stringify({ email: "user@example.com", password: "secret" }, null, 2)}</Pre>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ token: "gater-key-...", user: { id: "...", email: "user@example.com", name: "User", api_key: "gater-key-..." } }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="GET" path="/auth/me">
            <p className="text-sm text-muted-foreground">Get current user info.</p>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ id: "...", email: "user@example.com", name: "User", api_key: "gater-key-..." }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="POST" path="/auth/regenerate-key">
            <p className="text-sm text-muted-foreground">Regenerate your API key. The old key stops working immediately.</p>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ api_key: "gater-key-..." }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="POST" path="/auth/credential">
            <p className="text-sm text-muted-foreground">Save credential for a provider (key-value pairs stored as JSON).</p>
            <p className="text-xs font-medium mt-1">Request Body:</p>
            <Pre>{JSON.stringify({ provider: "doodstream", credentials: { api_key: "xxx" } }, null, 2)}</Pre>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ status: "saved" }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="GET" path="/auth/credentials">
            <p className="text-sm text-muted-foreground">List all stored credentials (without values).</p>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ credentials: [{ provider: "doodstream", updated_at: "..." }] }, null, 2)}</Pre>
          </Endpoint>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Providers</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Endpoint method="GET" path="/providers">
            <p className="text-sm text-muted-foreground">List all available providers with their capabilities.</p>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({
              providers: [
                { name: "doodstream", type: "video_host", label: "Video Host", anonymous_upload: false, remote_url_upload: true, has_api: true }
              ]
            }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="GET" path="/providers/{name}">
            <p className="text-sm text-muted-foreground">Get details for a specific provider.</p>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ name: "doodstream", type: "video_host", anonymous_upload: false, remote_url_upload: true, has_api: true }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="GET" path="/providers/{name}/credentials">
            <p className="text-sm text-muted-foreground">Get credential fields (with whether each is configured) for a provider.</p>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ provider: "doodstream", has_creds: true, fields: [{ key: "api_key", label: "API Key", has_value: true }] }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="PUT" path="/providers/{name}/credentials">
            <p className="text-sm text-muted-foreground">Update credentials for a provider.</p>
            <p className="text-xs font-medium mt-1">Request Body:</p>
            <Pre>{JSON.stringify({ values: { api_key: "xxx", folder_id: "123" } }, null, 2)}</Pre>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ status: "ok" }, null, 2)}</Pre>
          </Endpoint>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Tasks</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Endpoint method="POST" path="/upload/url">
            <p className="text-sm text-muted-foreground">Start an upload from a remote URL to selected providers.</p>
            <p className="text-xs font-medium mt-1">Request Body:</p>
            <Pre>{JSON.stringify({ url: "https://example.com/video.mp4", providers: ["doodstream", "vikingfiles"] }, null, 2)}</Pre>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ task_id: "uuid-here", status: "pending" }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="POST" path="/upload">
            <p className="text-sm text-muted-foreground">Upload a file directly to selected providers (multipart form).</p>
            <p className="text-xs font-medium mt-1">Form fields:</p>
            <ul className="text-sm list-disc list-inside text-muted-foreground">
              <li><Code>file</Code> (required) — the file to upload</li>
              <li><Code>providers[]</Code> (required) — provider names, repeat for each</li>
              <li><Code>title</Code> (optional) — display title</li>
            </ul>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ task_id: "uuid-here", status: "pending" }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="GET" path="/tasks">
            <p className="text-sm text-muted-foreground">List tasks. Supports pagination.</p>
            <p className="text-xs font-medium mt-1">Query params:</p>
            <ul className="text-sm list-disc list-inside text-muted-foreground">
              <li><Code>limit</Code> — max results (default 20, max 100)</li>
              <li><Code>offset</Code> — pagination offset</li>
            </ul>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ tasks: [{ id: "...", status: "completed", title: "...", file_name: "...", file_size: 12345, source_type: "remote_url", source_url: "...", created_at: "...", updated_at: "...", completed_at: "...", results: [{ provider: "doodstream", status: "completed", progress: 100, output_url: "...", file_code: "..." }] }], total: 1, limit: 20, offset: 0 }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="GET" path="/task/{id}">
            <p className="text-sm text-muted-foreground">Get a single task with all provider results.</p>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ id: "...", status: "completed", title: "...", file_name: "...", file_size: 12345, source_type: "remote_url", source_url: "...", created_at: "...", updated_at: "...", completed_at: "...", results: [{ provider: "doodstream", status: "completed", progress: 100, output_url: "...", file_code: "..." }] }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="GET" path="/task/{id}/progress">
            <p className="text-sm text-muted-foreground">SSE stream for real-time task progress. Returns Server-Sent Events until all providers complete or fail.</p>
            <p className="text-xs font-medium mt-1">Event format:</p>
            <Pre>{`data: {"id":"...","status":"processing","results":[{"provider":"doodstream","status":"uploading","progress":45,"output_url":"","file_code":"","error":""}]}\n\n`}</Pre>
            <p className="text-xs text-muted-foreground">
              Connect with <Code>EventSource</Code> in browser or any SSE client.
            </p>
          </Endpoint>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Settings</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Endpoint method="GET" path="/settings">
            <p className="text-sm text-muted-foreground">Get user settings.</p>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ default_providers: ["doodstream", "vikingfiles"], api_key: "gater-key-..." }, null, 2)}</Pre>
          </Endpoint>

          <Endpoint method="PUT" path="/settings">
            <p className="text-sm text-muted-foreground">Update user settings.</p>
            <p className="text-xs font-medium mt-1">Request Body:</p>
            <Pre>{JSON.stringify({ default_providers: ["doodstream", "vikingfiles"] }, null, 2)}</Pre>
            <p className="text-xs font-medium mt-1">Response:</p>
            <Pre>{JSON.stringify({ status: "ok" }, null, 2)}</Pre>
          </Endpoint>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Error Format</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-2">All error responses follow this format:</p>
          <Pre>{JSON.stringify({ error: "description of what went wrong" }, null, 2)}</Pre>
          <div className="mt-3 space-y-1 text-sm">
            <div className="flex gap-2"><Badge variant="outline" className="w-16 shrink-0">400</Badge> Bad request — invalid input</div>
            <div className="flex gap-2"><Badge variant="outline" className="w-16 shrink-0">401</Badge> Unauthorized — missing or invalid API key</div>
            <div className="flex gap-2"><Badge variant="outline" className="w-16 shrink-0">404</Badge> Not found</div>
            <div className="flex gap-2"><Badge variant="outline" className="w-16 shrink-0">409</Badge> Conflict — email already taken</div>
            <div className="flex gap-2"><Badge variant="outline" className="w-16 shrink-0">500</Badge> Internal server error</div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

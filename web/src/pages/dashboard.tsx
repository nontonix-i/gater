import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import * as api from "../lib/api"
import { Button } from "../components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../components/ui/card"
import { Badge } from "../components/ui/badge"
import { Skeleton } from "../components/ui/skeleton"
import {
  Film,
  HardDrive,
  CheckCircle2,
  XCircle,
  Clock,
} from "lucide-react"

export default function DashboardPage() {
  const navigate = useNavigate()
  const [providers, setProviders] = useState<api.Provider[]>([])
  const [recentTasks, setRecentTasks] = useState<api.Task[]>([])
  const [loading, setLoading] = useState(true)
  const [uploadUrl, setUploadUrl] = useState("")
  const [selectedProviders, setSelectedProviders] = useState<string[]>([])

  const remoteProviders = providers.filter(
    (p) => p.supports_remote_url
  )

  useEffect(() => {
    Promise.all([
      api.providers.list(),
      api.tasks.list({ limit: 5 }),
      api.settings.get(),
    ])
      .then(([provs, taskRes, s]) => {
        setProviders(Array.isArray(provs) ? provs : [])
        setRecentTasks(Array.isArray(taskRes.tasks) ? taskRes.tasks : [])
        if (s.default_providers?.length > 0) {
          setSelectedProviders(s.default_providers)
        }
      })
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (remoteProviders.length > 0 && selectedProviders.length === 0) {
      setSelectedProviders(remoteProviders.slice(0, 3).map((p) => p.name))
    }
  }, [remoteProviders, selectedProviders.length])

  const toggleProvider = (name: string) => {
    setSelectedProviders((prev) =>
      prev.includes(name)
        ? prev.filter((p) => p !== name)
        : [...prev, name]
    )
  }

  const handleQuickUpload = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!uploadUrl.trim()) return

    if (selectedProviders.length === 0) {
      alert("Select at least one provider")
      return
    }

    try {
      const res = await api.tasks.create({
        url: uploadUrl,
        providers: selectedProviders,
      })
      navigate(`/tasks/${res.task_id}`)
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Upload failed")
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <div className="grid gap-4 md:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">
          Monitor uploads and manage providers
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">
              Video Hosts
            </CardTitle>
            <Film className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {providers.filter((p) => p.type === "video_host").length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Storage</CardTitle>
            <HardDrive className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {providers.filter((p) => p.type === "storage").length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Recent</CardTitle>
            <CheckCircle2 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{recentTasks.length}</div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Quick Upload</CardTitle>
          <CardDescription>
            Upload from URL to selected providers
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleQuickUpload} className="space-y-3">
            <input
              type="url"
              value={uploadUrl}
              onChange={(e) => setUploadUrl(e.target.value)}
              placeholder="https://example.com/video.mp4"
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              required
            />
            {remoteProviders.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {remoteProviders.map((p) => (
                  <label
                    key={p.name}
                    className={`flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-sm cursor-pointer transition-colors ${
                      selectedProviders.includes(p.name)
                        ? "bg-primary text-primary-foreground border-primary"
                        : "bg-background hover:bg-accent"
                    }`}
                  >
                    <input
                      type="checkbox"
                      checked={selectedProviders.includes(p.name)}
                      onChange={() => toggleProvider(p.name)}
                      className="sr-only"
                    />
                    <span className="capitalize">{p.name}</span>
                  </label>
                ))}
              </div>
            )}
            {selectedProviders.length === 0 && (
              <p className="text-xs text-destructive">
                Select at least one provider
              </p>
            )}
            <Button type="submit" disabled={selectedProviders.length === 0}>
              Upload
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Recent Tasks</CardTitle>
          <CardDescription>
            Your most recent upload activities
          </CardDescription>
        </CardHeader>
        <CardContent>
          {recentTasks.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No tasks yet. Upload your first file!
            </p>
          ) : (
            <div className="space-y-3">
              {recentTasks.map((task) => (
                <div
                  key={task.id}
                  className="flex items-center justify-between rounded-lg border p-3 cursor-pointer hover:bg-accent/50 transition-colors"
                  onClick={() => navigate(`/tasks/${task.id}`)}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div>
                      {task.status === "completed" ? (
                        <CheckCircle2 className="h-5 w-5 text-green-500" />
                      ) : task.status === "failed" ? (
                        <XCircle className="h-5 w-5 text-red-500" />
                      ) : (
                        <Clock className="h-5 w-5 text-yellow-500" />
                      )}
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm font-medium truncate">
                        {task.title || task.file_name || task.source_url}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {new Date(task.created_at).toLocaleDateString()} ·{" "}
                        {task.results?.length || 0} providers
                      </p>
                    </div>
                  </div>
                  <Badge variant="secondary">{task.status}</Badge>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

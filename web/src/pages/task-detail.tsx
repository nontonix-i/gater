import { useEffect, useState, useRef } from "react"
import { useParams, useNavigate } from "react-router-dom"
import * as api from "../lib/api"
import type { ProgressEvent } from "../lib/api"
import { Button } from "../components/ui/button"
import { Badge } from "../components/ui/badge"
import { Skeleton } from "../components/ui/skeleton"
import { Separator } from "../components/ui/separator"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../components/ui/card"
import {
  ArrowLeft,
  ExternalLink,
  CheckCircle2,
  XCircle,
  Clock,
  Upload,
} from "lucide-react"

const resultStatusIcon: Record<string, React.ReactNode> = {
  completed: <CheckCircle2 className="h-5 w-5 text-green-500" />,
  failed: <XCircle className="h-5 w-5 text-red-500" />,
  uploading: <Upload className="h-5 w-5 text-blue-500" />,
  pending: <Clock className="h-5 w-5 text-yellow-500" />,
}

export default function TaskDetailPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [task, setTask] = useState<api.Task | null>(null)
  const [live, setLive] = useState<ProgressEvent | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")
  const doneRef = useRef(false)

  useEffect(() => {
    if (!id) return

    api.tasks.get(id).then((t) => {
      setTask(t)
      setLoading(false)

      const allDone = t.results?.every(
        (r) => r.status === "completed" || r.status === "failed"
      )
      if (allDone) return

      const unsub = api.subscribeProgress(id, (evt) => {
        setLive(evt)
        if (evt.done && !doneRef.current) {
          doneRef.current = true
          api.tasks.get(id).then(setTask)
        }
      }, setError)

      return unsub
    }).catch(() => {
      setLoading(false)
    })
  }, [id])

  const results = live?.results || task?.results || []
  const status = live?.status || task?.status || ""
  const displayId = live?.id || task?.id || id

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }

  if (!task) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">Task not found</p>
        <Button variant="link" onClick={() => navigate("/tasks")}>
          Back to tasks
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate("/tasks")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {task.title || task.file_name || "Task"}
          </h1>
          <p className="text-sm text-muted-foreground font-mono">{displayId}</p>
        </div>
        <Badge
          variant={
            status === "completed"
              ? "default"
              : status === "failed"
                ? "destructive"
                : "secondary"
          }
          className="ml-auto"
        >
          {status}
        </Badge>
      </div>

      {error && (
        <p className="text-sm text-destructive">{error}</p>
      )}

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Source</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm truncate">
              {task.source_url || task.file_path || "-"}
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              Type: {task.source_type} · Size:{" "}
              {task.file_size
                ? `${(task.file_size / 1024 / 1024).toFixed(1)} MB`
                : "-"}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Timeline</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm">
              Created: {new Date(task.created_at).toLocaleString()}
            </p>
            {task.completed_at && (
              <p className="text-sm">
                Completed: {new Date(task.completed_at).toLocaleString()}
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      <Separator />

      <div>
        <h2 className="text-lg font-semibold mb-4">
          Provider Results ({results.length})
        </h2>
        <div className="space-y-3">
          {results.map((result) => {
            const isLive = live?.results?.some((r) => r.provider === result.provider)

            return (
              <Card key={result.provider}>
                <CardContent className="pt-6">
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-3 flex-1 min-w-0">
                      {resultStatusIcon[result.status] || (
                        <Clock className="h-5 w-5 text-muted-foreground shrink-0" />
                      )}
                      <div className="min-w-0 space-y-1 flex-1">
                        <p className="font-medium capitalize">{result.provider}</p>
                        <Badge variant="outline">{result.status}</Badge>

                        {result.status === "uploading" && (
                          <div className="w-full max-w-xs mt-2">
                            <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
                              <div
                                className="h-full rounded-full bg-primary transition-all duration-300"
                                style={{ width: `${result.progress}%` }}
                              />
                            </div>
                            <p className="text-xs text-muted-foreground mt-0.5">
                              {result.progress}%
                            </p>
                          </div>
                        )}

                        {result.error_message && (
                          <p className="text-sm text-destructive">{result.error_message}</p>
                        )}
                        {result.error && !result.error_message && (
                          <p className="text-sm text-destructive">{result.error}</p>
                        )}

                        {result.output_url && (
                          <a
                            href={result.output_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="flex items-center gap-1 text-sm text-primary hover:underline truncate"
                          >
                            <ExternalLink className="h-3 w-3 shrink-0" />
                            {result.output_url}
                          </a>
                        )}

                        {result.file_code && (
                          <p className="text-xs text-muted-foreground">
                            File code: <span className="font-mono">{result.file_code}</span>
                          </p>
                        )}

                        {result.provider_file_name && (
                          <p className="text-xs text-muted-foreground">
                            Remote name: {result.provider_file_name}
                            {result.provider_file_size > 0 && (
                              <> · {(result.provider_file_size / 1024 / 1024).toFixed(1)} MB</>
                            )}
                          </p>
                        )}

                        {result.started_at && (
                          <p className="text-xs text-muted-foreground">
                            Started: {new Date(result.started_at).toLocaleString()}
                          </p>
                        )}
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      </div>
    </div>
  )
}

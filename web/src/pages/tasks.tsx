import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import * as api from "../lib/api"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table"
import { Badge } from "../components/ui/badge"
import { Skeleton } from "../components/ui/skeleton"
import { Plus, Search } from "lucide-react"

export default function TasksPage() {
  const navigate = useNavigate()
  const [taskList, setTaskList] = useState<api.Task[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState("")

  const loadTasks = async () => {
    setLoading(true)
    try {
      const res = await api.tasks.list({ limit: 50 })
      setTaskList(res.tasks ?? [])
      setTotal(res.total ?? 0)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadTasks()
  }, [])

  const filtered = search
    ? taskList.filter(
        (t) =>
          t.title?.toLowerCase().includes(search.toLowerCase()) ||
          t.file_name?.toLowerCase().includes(search.toLowerCase()) ||
          t.source_url?.toLowerCase().includes(search.toLowerCase())
      )
    : taskList

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Tasks</h1>
          <p className="text-muted-foreground">
            {total} total upload tasks
          </p>
        </div>
        <Button onClick={() => navigate("/")}>
          <Plus className="h-4 w-4 mr-2" />
          New Upload
        </Button>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="Search tasks..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-9"
        />
      </div>

      {loading ? (
        <div className="space-y-2">
          {[1, 2, 3, 4, 5].map((i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-muted-foreground">No tasks found</p>
        </div>
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Source</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Providers</TableHead>
                <TableHead>Date</TableHead>
                <TableHead className="text-right">Results</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((task) => {
                const completedCount = task.results?.filter(
                  (r) => r.status === "completed"
                ).length
                const failedCount = task.results?.filter(
                  (r) => r.status === "failed"
                ).length

                return (
                  <TableRow
                    key={task.id}
                    className="cursor-pointer"
                    onClick={() => navigate(`/tasks/${task.id}`)}
                  >
                    <TableCell className="font-medium max-w-[200px] truncate">
                      {task.title || task.file_name || task.source_url || "-"}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={
                          task.status === "completed"
                            ? "default"
                            : task.status === "failed"
                              ? "destructive"
                              : "secondary"
                        }
                      >
                        {task.status}
                      </Badge>
                    </TableCell>
                    <TableCell>{task.results?.length || 0}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {new Date(task.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <span className="text-green-600 dark:text-green-400">
                        {completedCount}
                      </span>
                      {failedCount > 0 && (
                        <>
                          {" / "}
                          <span className="text-red-600 dark:text-red-400">
                            {failedCount}
                          </span>
                        </>
                      )}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  )
}

import { useEffect, useState } from "react"
import { Link, Outlet, useNavigate, useLocation } from "react-router-dom"
import { useAuth } from "../hooks/use-auth"
import { useTheme } from "../hooks/use-theme"
import { Avatar, AvatarFallback } from "../components/ui/avatar"
import {
  CloudUpload,
  LayoutDashboard,
  List,
  Moon,
  Sun,
  LogOut,
  Settings,
  BookOpen,
  PanelLeftClose,
  PanelLeft,
} from "lucide-react"

const navItems = [
  { to: "/", icon: LayoutDashboard, label: "Dashboard" },
  { to: "/tasks", icon: List, label: "Tasks" },
  { to: "/docs", icon: BookOpen, label: "API Docs" },
]

export default function Layout() {
  const { user, logout } = useAuth()
  const { theme, setTheme } = useTheme()
  const navigate = useNavigate()
  const location = useLocation()
  const collapsed = localStorage.getItem("gater_sidebar") === "collapsed"

  const isActive = (path: string) => {
    if (path === "/") return location.pathname === "/"
    return location.pathname.startsWith(path)
  }

  const toggleSidebar = () => {
    if (collapsed) {
      localStorage.removeItem("gater_sidebar")
    } else {
      localStorage.setItem("gater_sidebar", "collapsed")
    }
    window.dispatchEvent(new Event("storage"))
    // force re-render by using a custom event
    window.dispatchEvent(new Event("sidebartoggle"))
  }

  // use a key state to force re-render on toggle
  const [sidebarOpen, setSidebarOpen] = useState(!collapsed)
  useEffect(() => {
    const handler = () => setSidebarOpen(localStorage.getItem("gater_sidebar") !== "collapsed")
    window.addEventListener("sidebartoggle", handler)
    window.addEventListener("storage", handler)
    return () => {
      window.removeEventListener("sidebartoggle", handler)
      window.removeEventListener("storage", handler)
    }
  }, [])

  const handleLogout = () => {
    logout()
    navigate("/login")
  }

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside
        className={`${
          sidebarOpen ? "w-56" : "w-14"
        } shrink-0 border-r bg-muted/30 flex flex-col transition-[width] duration-200`}
      >
        {/* Logo + collapse button */}
        <div className="flex items-center h-14 border-b px-3 gap-2">
          {sidebarOpen && (
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <CloudUpload className="h-5 w-5 text-primary shrink-0" />
              <span className="font-semibold text-sm truncate">Gater</span>
            </div>
          )}
          {!sidebarOpen && <CloudUpload className="h-5 w-5 text-primary mx-auto" />}
          <button
            onClick={toggleSidebar}
            className="p-1.5 rounded-md hover:bg-accent text-muted-foreground hover:text-foreground transition-colors shrink-0"
            title={sidebarOpen ? "Collapse sidebar" : "Expand sidebar"}
          >
            {sidebarOpen ? <PanelLeftClose className="h-4 w-4" /> : <PanelLeft className="h-4 w-4" />}
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 flex flex-col gap-0.5 p-2">
          {navItems.map((item) => (
            <Link
              key={item.to}
              to={item.to}
              className={`flex items-center gap-2.5 px-2.5 py-2 rounded-md text-sm transition-colors ${
                isActive(item.to)
                  ? "bg-primary/10 text-primary font-medium"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              } ${!sidebarOpen ? "justify-center px-0" : ""}`}
              title={!sidebarOpen ? item.label : undefined}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              {sidebarOpen && item.label}
            </Link>
          ))}
        </nav>

        {/* Bottom section */}
        <div className={`border-t p-2 space-y-1 ${!sidebarOpen ? "flex flex-col items-center" : ""}`}>
          <Link
            to="/settings"
            className={`flex items-center gap-2.5 px-2.5 py-2 rounded-md text-sm transition-colors ${
              isActive("/settings")
                ? "bg-primary/10 text-primary font-medium"
                : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            } ${!sidebarOpen ? "justify-center px-0" : ""}`}
            title={!sidebarOpen ? "Settings" : undefined}
          >
            <Settings className="h-4 w-4 shrink-0" />
            {sidebarOpen && "Settings"}
          </Link>

          <div className={`flex items-center gap-2 px-2.5 py-2 ${!sidebarOpen ? "flex-col" : ""}`}>
            <Avatar className="h-7 w-7 shrink-0">
              <AvatarFallback className="text-xs">
                {user?.name?.charAt(0)?.toUpperCase() || "U"}
              </AvatarFallback>
            </Avatar>
            {sidebarOpen && (
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">{user?.name || user?.email}</p>
              </div>
            )}
            <div className="flex items-center gap-0.5">
              <button
                onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
                className="p-1.5 rounded-md hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
                title="Toggle theme"
              >
                {theme === "dark" ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
              </button>
              <button
                onClick={handleLogout}
                className="p-1.5 rounded-md hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
                title="Logout"
              >
                <LogOut className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 bg-background min-w-0 overflow-y-auto">
        <div className="px-6 py-8">
          <Outlet />
        </div>
      </main>
    </div>
  )
}

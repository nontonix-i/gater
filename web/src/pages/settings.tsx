import { useEffect, useState } from "react"
import * as api from "../lib/api"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card"
import { Badge } from "../components/ui/badge"
import { Skeleton } from "../components/ui/skeleton"
import { Separator } from "../components/ui/separator"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { Save, Key, RefreshCw, Check, Eye, EyeOff, Upload } from "lucide-react"

export default function SettingsPage() {
  const [providers, setProviders] = useState<api.Provider[]>([])
  const [settings, setSettings] = useState<api.Settings | null>(null)
  const [defaultProvs, setDefaultProvs] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)

  const [selectedProvider, setSelectedProvider] = useState<string>("")
  const [credFields, setCredFields] = useState<{ key: string; label: string; has_value: boolean }[]>([])
  const [credValues, setCredValues] = useState<Record<string, string>>({})
  const [credSaving, setCredSaving] = useState(false)
  const [credHasCreds, setCredHasCreds] = useState(false)

  const [regenerating, setRegenerating] = useState(false)
  const [newKey, setNewKey] = useState<string | null>(null)
  const [showKey, setShowKey] = useState(false)

  const [bulkInput, setBulkInput] = useState("")
  const [bulkTarget, setBulkTarget] = useState("")

  useEffect(() => {
    Promise.all([
      api.providers.list(),
      api.settings.get(),
    ]).then(([provs, s]) => {
      setProviders(provs.filter(p => p.has_api))
      setSettings(s)
      setDefaultProvs(s.default_providers ?? [])
    }).finally(() => setLoading(false))
  }, [])

  const loadProviderCreds = async (name: string) => {
    setSelectedProvider(name)
    setCredValues({})
    try {
      const cred = await api.providers.getCredentials(name)
      setCredFields(cred.fields)
      setCredHasCreds(cred.has_creds)
    } catch {
      setCredFields([])
      setCredHasCreds(false)
    }
  }

  const toggleDefaultProvider = (name: string) => {
    setDefaultProvs(prev =>
      prev.includes(name) ? prev.filter(p => p !== name) : [...prev, name]
    )
  }

  const saveSettings = async () => {
    setSaving(true)
    setSaved(false)
    try {
      await api.settings.update({ default_providers: defaultProvs })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch { }
    setSaving(false)
  }

  const saveCredential = async () => {
    if (!selectedProvider) return
    setCredSaving(true)
    try {
      const filled: Record<string, string> = {}
      for (const f of credFields) {
        if (credValues[f.key]) {
          filled[f.key] = credValues[f.key]
        }
      }
      await api.providers.saveCredentials(selectedProvider, filled)
      await loadProviderCreds(selectedProvider)
    } catch { }
    setCredSaving(false)
  }

  const handleBulkImport = async () => {
    if (!bulkTarget || !bulkInput.trim()) return
    const lines = bulkInput.trim().split("\n")
    const values: Record<string, string> = {}
    for (const line of lines) {
      const eqIdx = line.indexOf("=")
      if (eqIdx === -1) continue
      const key = line.slice(0, eqIdx).trim()
      const val = line.slice(eqIdx + 1).trim()
      if (key && val) values[key] = val
    }
    if (Object.keys(values).length === 0) return
    setCredSaving(true)
    try {
      await api.providers.saveCredentials(bulkTarget, values)
      await loadProviderCreds(bulkTarget)
      setBulkInput("")
    } catch { }
    setCredSaving(false)
  }

  const handleRegenerateKey = async () => {
    if (!confirm("Regenerate API key? Your current key will stop working immediately.")) return
    setRegenerating(true)
    try {
      const res = await api.settings.regenerateKey()
      setNewKey(res.api_key)
    } catch { }
    setRegenerating(false)
  }

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">Manage credentials, defaults, and API key</p>
      </div>

      {/* API Key */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="h-4 w-4" />
            API Key
          </CardTitle>
          <CardDescription>Your API key for programmatic access to Gater</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {newKey ? (
            <div className="flex items-center gap-2 p-3 rounded-lg border bg-amber-50 dark:bg-amber-950/20">
              <code className="text-sm break-all font-mono flex-1">
                {showKey ? newKey : "•".repeat(40)}
              </code>
              <Button variant="ghost" size="icon" onClick={() => setShowKey(!showKey)}>
                {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2 p-3 rounded-lg border bg-muted/30">
              <code className="text-sm break-all font-mono flex-1">
                {showKey ? settings?.api_key : "•".repeat(40)}
              </code>
              <Button variant="ghost" size="icon" onClick={() => setShowKey(!showKey)}>
                {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </Button>
            </div>
          )}
          <Button variant="outline" size="sm" onClick={handleRegenerateKey} disabled={regenerating}>
            <RefreshCw className={`h-4 w-4 mr-2 ${regenerating ? "animate-spin" : ""}`} />
            Regenerate
          </Button>
        </CardContent>
      </Card>

      {/* Default Providers */}
      <Card>
        <CardHeader>
          <CardTitle>Default Providers</CardTitle>
          <CardDescription>
            Select which providers are pre-checked when uploading
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap gap-2">
            {providers.map(p => (
              <label
                key={p.name}
                className={`flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-sm cursor-pointer transition-colors ${
                  defaultProvs.includes(p.name)
                    ? "bg-primary text-primary-foreground border-primary"
                    : "bg-background hover:bg-accent"
                }`}
              >
                <input
                  type="checkbox"
                  checked={defaultProvs.includes(p.name)}
                  onChange={() => toggleDefaultProvider(p.name)}
                  className="sr-only"
                />
                <span className="capitalize">{p.name}</span>
              </label>
            ))}
          </div>
          <div className="flex items-center gap-2">
            <Button size="sm" onClick={saveSettings} disabled={saving}>
              {saved ? (
                <><Check className="h-4 w-4 mr-1" /> Saved</>
              ) : (
                <><Save className="h-4 w-4 mr-1" /> Save</>
              )}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Provider Credentials */}
      <Card>
        <CardHeader>
          <CardTitle>Provider Credentials</CardTitle>
          <CardDescription>Set API keys, tokens, and login for each provider</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <Label>Select Provider</Label>
            <Select value={selectedProvider} onValueChange={(v) => v && loadProviderCreds(v)}>
              <SelectTrigger className="w-full capitalize">
                <SelectValue placeholder="Choose a provider..." />
              </SelectTrigger>
              <SelectContent>
                {providers.map(p => (
                  <SelectItem key={p.name} value={p.name} className="capitalize">
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {selectedProvider && (
            <>
              <Separator />

              {/* Manual input */}
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <span className="font-medium capitalize">{selectedProvider}</span>
                  <Badge variant={credHasCreds ? "default" : "secondary"}>
                    {credHasCreds ? "Configured" : "Not set"}
                  </Badge>
                </div>
                {credFields.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    This provider doesn't require credentials.
                  </p>
                ) : (
                  <div className="space-y-2">
                    {credFields.map(f => (
                      <div key={f.key}>
                        <Label className="text-xs">{f.label}</Label>
                        <Input
                          type="password"
                          placeholder={f.has_value ? "Leave empty to keep existing" : `Enter ${f.label}`}
                          value={credValues[f.key] ?? ""}
                          onChange={e => setCredValues(prev => ({ ...prev, [f.key]: e.target.value }))}
                        />
                      </div>
                    ))}
                    <Button size="sm" variant="outline" onClick={saveCredential} disabled={credSaving}>
                      <Save className="h-4 w-4 mr-1" />
                      {credSaving ? "Saving..." : "Save Credentials"}
                    </Button>
                  </div>
                )}
              </div>

              <Separator />

              {/* Bulk import */}
              <div className="space-y-2">
                <Label>Bulk Import (env format)</Label>
                <p className="text-xs text-muted-foreground">
                  Paste credentials in KEY=VALUE format, one per line.
                </p>
                <textarea
                  value={bulkInput}
                  onChange={e => setBulkInput(e.target.value)}
                  placeholder={`api_key=your_key_here\nfolder_id=12345`}
                  rows={4}
                  className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={handleBulkImport}
                    disabled={credSaving || !bulkInput.trim()}
                  >
                    <Upload className="h-4 w-4 mr-1" />
                    {credSaving ? "Importing..." : "Import"}
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

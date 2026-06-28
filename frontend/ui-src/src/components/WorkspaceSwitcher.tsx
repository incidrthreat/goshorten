import { useEffect, useRef, useState } from 'react'
import { Building2, Check, ChevronsUpDown, Plus } from 'lucide-react'
import { workspaces as workspacesApi, type Workspace } from '../api/client'

// WorkspaceSwitcher shows the active workspace and lets the user switch between
// workspaces they can access or create a new one (Phase 14.8). Switching persists
// the choice (per-session on the server + localStorage for the request header) and
// reloads so every page refetches data scoped to the new tenant.
export default function WorkspaceSwitcher() {
  const [list, setList] = useState<Workspace[]>([])
  const [activeId, setActiveId] = useState<number | null>(null)
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let cancelled = false
    workspacesApi
      .list()
      .then((res) => {
        if (cancelled) return
        setList(res.workspaces)
        setActiveId(res.activeWorkspaceId)
        if (res.activeWorkspaceId && !localStorage.getItem('workspaceId')) {
          localStorage.setItem('workspaceId', String(res.activeWorkspaceId))
        }
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [])

  const active = list.find((w) => w.id === activeId)

  const switchTo = async (id: number) => {
    if (id === activeId || busy) return
    setBusy(true)
    try {
      await workspacesApi.switch(id)
      localStorage.setItem('workspaceId', String(id))
      window.location.reload()
    } catch {
      setBusy(false)
    }
  }

  const createWorkspace = async () => {
    const name = window.prompt('New workspace name')?.trim()
    if (!name) return
    setBusy(true)
    try {
      const ws = await workspacesApi.create(name)
      await workspacesApi.switch(ws.id)
      localStorage.setItem('workspaceId', String(ws.id))
      window.location.reload()
    } catch (e) {
      setBusy(false)
      alert((e as Error).message)
    }
  }

  if (list.length === 0) {
    return null
  }

  return (
    <div className="relative mr-2" ref={ref}>
      <button
        onClick={() => setOpen((o) => !o)}
        disabled={busy}
        className="flex items-center gap-2 max-w-[12rem] px-3 py-1.5 rounded-lg text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
        title="Switch workspace"
      >
        <Building2 className="w-4 h-4 shrink-0 text-blue-500" />
        <span className="truncate">{active ? active.name : 'Workspace'}</span>
        <ChevronsUpDown className="w-4 h-4 shrink-0 text-gray-400" />
      </button>

      {open && (
        <div className="absolute right-0 mt-2 w-64 z-20 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-lg py-1">
          <div className="px-3 py-1.5 text-xs font-semibold uppercase tracking-wide text-gray-400">
            Workspaces
          </div>
          <ul className="max-h-64 overflow-auto">
            {list.map((ws) => (
              <li key={ws.id}>
                <button
                  onClick={() => switchTo(ws.id)}
                  className="flex items-center justify-between w-full px-3 py-2 text-sm text-left text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800"
                >
                  <span className="truncate">
                    {ws.name}
                    {!ws.isOwner && (
                      <span className="ml-2 text-xs text-gray-400">shared</span>
                    )}
                  </span>
                  {ws.id === activeId && <Check className="w-4 h-4 text-blue-500 shrink-0" />}
                </button>
              </li>
            ))}
          </ul>
          <div className="border-t border-gray-200 dark:border-gray-700 mt-1 pt-1">
            <button
              onClick={createWorkspace}
              className="flex items-center gap-2 w-full px-3 py-2 text-sm text-left text-blue-600 dark:text-blue-400 hover:bg-gray-100 dark:hover:bg-gray-800"
            >
              <Plus className="w-4 h-4" />
              New workspace
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

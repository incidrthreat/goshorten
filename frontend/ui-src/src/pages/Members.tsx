import { useCallback, useEffect, useState } from 'react'
import {
  members as membersApi,
  invitations as invitationsApi,
  type Member,
  type Invitation,
} from '../api/client'
import { Crown, Mail, Trash2, RefreshCw, Copy, UserPlus, Check } from 'lucide-react'

interface MembersProps {
  user: { id: string; email: string; role: string }
}

const ROLE_LABELS: Record<string, string> = {
  owner: 'Owner',
  admin: 'Admin',
  member: 'Member',
  viewer: 'Viewer',
}

function roleBadgeClass(role: string): string {
  switch (role) {
    case 'owner':
      return 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200'
    case 'admin':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
    case 'viewer':
      return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
    default:
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-200'
  }
}

export default function Members({ user }: MembersProps) {
  const workspaceId = Number(localStorage.getItem('workspaceId') || 0)
  const [members, setMembers] = useState<Member[]>([])
  const [invites, setInvites] = useState<Invitation[]>([])
  const [yourRole, setYourRole] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState('member')
  const [inviting, setInviting] = useState(false)
  const [copiedId, setCopiedId] = useState<number | null>(null)

  const canManage = yourRole === 'owner' || yourRole === 'admin' || user.role === 'admin'
  const isOwner = yourRole === 'owner' || user.role === 'admin'

  const load = useCallback(async () => {
    if (!workspaceId) {
      setError('No active workspace selected.')
      setLoading(false)
      return
    }
    setLoading(true)
    setError('')
    try {
      const res = await membersApi.list(workspaceId)
      setMembers(res.members || [])
      setInvites(res.pendingInvites || [])
      setYourRole(res.yourRole)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load members')
    } finally {
      setLoading(false)
    }
  }, [workspaceId])

  useEffect(() => {
    load()
  }, [load])

  const flash = (msg: string) => {
    setNotice(msg)
    setTimeout(() => setNotice(''), 4000)
  }

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault()
    setInviting(true)
    setError('')
    try {
      const inv = await invitationsApi.create(workspaceId, inviteEmail.trim(), inviteRole)
      setInviteEmail('')
      setInviteRole('member')
      if (inv.acceptUrl) {
        await navigator.clipboard?.writeText(inv.acceptUrl).catch(() => {})
        flash(`Invitation created. Accept link copied to clipboard.`)
      } else {
        flash('Invitation sent.')
      }
      load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send invitation')
    } finally {
      setInviting(false)
    }
  }

  const changeRole = async (m: Member, role: string) => {
    setError('')
    try {
      await membersApi.updateRole(workspaceId, m.userId, role)
      load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to change role')
    }
  }

  const removeMember = async (m: Member) => {
    const self = String(m.userId) === user.id
    if (!confirm(self ? 'Leave this workspace?' : `Remove ${m.email} from this workspace?`)) return
    setError('')
    try {
      await membersApi.remove(workspaceId, m.userId)
      if (self) {
        localStorage.removeItem('workspaceId')
        window.location.href = '/'
        return
      }
      load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove member')
    }
  }

  const makeOwner = async (m: Member) => {
    if (!confirm(`Transfer ownership to ${m.email}? You will become an admin.`)) return
    setError('')
    try {
      await membersApi.transferOwnership(workspaceId, m.userId)
      flash(`Ownership transferred to ${m.email}.`)
      load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to transfer ownership')
    }
  }

  const resendInvite = async (inv: Invitation) => {
    setError('')
    try {
      const updated = await invitationsApi.resend(workspaceId, inv.id)
      if (updated.acceptUrl) {
        await navigator.clipboard?.writeText(updated.acceptUrl).catch(() => {})
        flash('Invitation resent. New accept link copied to clipboard.')
      } else {
        flash('Invitation resent.')
      }
      load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to resend invitation')
    }
  }

  const revokeInvite = async (inv: Invitation) => {
    if (!confirm(`Revoke the invitation to ${inv.email}?`)) return
    setError('')
    try {
      await invitationsApi.revoke(workspaceId, inv.id)
      load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to revoke invitation')
    }
  }

  const copyLink = async (inv: Invitation) => {
    if (!inv.acceptUrl) return
    await navigator.clipboard?.writeText(inv.acceptUrl).catch(() => {})
    setCopiedId(inv.id)
    setTimeout(() => setCopiedId(null), 2000)
  }

  return (
    <div className="space-y-6">
      {notice && (
        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-700 dark:bg-green-900/20 dark:border-green-800 dark:text-green-300">
          {notice}
        </div>
      )}
      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
          {error}
        </div>
      )}

      {/* Invite form (managers only) */}
      {canManage && (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-5">
          <h3 className="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
            <UserPlus className="w-4 h-4" /> Invite a member
          </h3>
          <form onSubmit={handleInvite} className="flex flex-col sm:flex-row gap-3">
            <input
              type="email"
              required
              placeholder="teammate@example.com"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
              className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none text-sm"
            />
            <select
              value={inviteRole}
              onChange={(e) => setInviteRole(e.target.value)}
              className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none text-sm"
            >
              <option value="admin">Admin</option>
              <option value="member">Member</option>
              <option value="viewer">Viewer</option>
            </select>
            <button
              type="submit"
              disabled={inviting}
              className="flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors text-sm font-medium"
            >
              <Mail className="w-4 h-4" />
              {inviting ? 'Sending...' : 'Send invite'}
            </button>
          </form>
          <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
            If email isn't configured, the accept link is copied to your clipboard so you can share it directly.
          </p>
        </div>
      )}

      {/* Members table */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">Members</h3>
          {yourRole && (
            <span className="text-xs text-gray-500 dark:text-gray-400">
              Your role: <span className={`ml-1 px-2 py-0.5 rounded ${roleBadgeClass(yourRole)}`}>{ROLE_LABELS[yourRole] || yourRole}</span>
            </span>
          )}
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Member</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Role</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Joined</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-800">
              {loading ? (
                <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-500 dark:text-gray-400">Loading...</td></tr>
              ) : members.length === 0 ? (
                <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-500 dark:text-gray-400">No members.</td></tr>
              ) : (
                members.map((m) => {
                  const self = String(m.userId) === user.id
                  return (
                    <tr key={m.userId} className="hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
                      <td className="px-4 py-3">
                        <div className="text-sm font-medium text-gray-900 dark:text-gray-100">
                          {m.email}
                          {self && <span className="ml-2 text-xs text-gray-400">(you)</span>}
                        </div>
                        {m.name && <div className="text-xs text-gray-500 dark:text-gray-400">{m.name}</div>}
                      </td>
                      <td className="px-4 py-3">
                        {canManage && m.role !== 'owner' ? (
                          <select
                            value={m.role}
                            onChange={(e) => changeRole(m, e.target.value)}
                            className="px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                          >
                            <option value="admin">Admin</option>
                            <option value="member">Member</option>
                            <option value="viewer">Viewer</option>
                          </select>
                        ) : (
                          <span className={`px-2 py-0.5 text-xs rounded ${roleBadgeClass(m.role)}`}>
                            {ROLE_LABELS[m.role] || m.role}
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                        {m.joinedAt ? new Date(m.joinedAt).toLocaleDateString() : '—'}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center justify-end gap-2">
                          {isOwner && m.role !== 'owner' && (
                            <button
                              onClick={() => makeOwner(m)}
                              className="flex items-center gap-1 text-xs px-2 py-1 border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-800 text-gray-600 dark:text-gray-400 transition-colors"
                              title="Transfer ownership"
                            >
                              <Crown className="w-3.5 h-3.5" /> Make owner
                            </button>
                          )}
                          {(canManage || self) && m.role !== 'owner' && (
                            <button
                              onClick={() => removeMember(m)}
                              className="p-1.5 rounded text-gray-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                              title={self ? 'Leave workspace' : 'Remove member'}
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Pending invitations (managers only) */}
      {canManage && invites.length > 0 && (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
          <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
            <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">Pending invitations</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Email</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Role</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Expires</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-800">
                {invites.map((inv) => (
                  <tr key={inv.id} className="hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
                    <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-100">{inv.email}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-0.5 text-xs rounded ${roleBadgeClass(inv.role)}`}>
                        {ROLE_LABELS[inv.role] || inv.role}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                      {inv.expiresAt ? new Date(inv.expiresAt).toLocaleDateString() : '—'}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-2">
                        {inv.acceptUrl && (
                          <button
                            onClick={() => copyLink(inv)}
                            className="p-1.5 rounded text-gray-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors"
                            title="Copy accept link"
                          >
                            {copiedId === inv.id ? <Check className="w-3.5 h-3.5 text-green-600" /> : <Copy className="w-3.5 h-3.5" />}
                          </button>
                        )}
                        <button
                          onClick={() => resendInvite(inv)}
                          className="p-1.5 rounded text-gray-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors"
                          title="Resend invitation"
                        >
                          <RefreshCw className="w-3.5 h-3.5" />
                        </button>
                        <button
                          onClick={() => revokeInvite(inv)}
                          className="p-1.5 rounded text-gray-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                          title="Revoke invitation"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}

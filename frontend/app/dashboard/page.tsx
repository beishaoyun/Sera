'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useAuthStore } from '@/lib/store'
import { serverService, deploymentService } from '@/lib/api'

export default function DashboardPage() {
  const router = useRouter()
  const { user, isAuthenticated, logout } = useAuthStore()
  const [servers, setServers] = useState<any[]>([])
  const [deployments, setDeployments] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login')
      return
    }

    loadData()
  }, [isAuthenticated])

  const loadData = async () => {
    try {
      const [serversRes, deploymentsRes] = await Promise.all([
        serverService.list(),
        deploymentService.list(),
      ])
      setServers(serversRes.data.servers || [])
      setDeployments(deploymentsRes.data.servers || [])
    } catch (error) {
      console.error('Failed to load data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleLogout = () => {
    logout()
    router.push('/')
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-white text-lg">加载中...</div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-slate-900">
      {/* Header */}
      <header className="border-b border-slate-700 bg-slate-800">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <Link href="/dashboard" className="text-xl font-bold text-white">
              ServerMind
            </Link>
            <div className="flex items-center gap-4">
              <span className="text-slate-300 text-sm">{user?.email}</span>
              <button
                onClick={handleLogout}
                className="px-4 py-2 text-sm text-slate-300 hover:text-white transition"
              >
                退出
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-8">
        {/* Stats */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
          <Card className="bg-slate-800 border-slate-700">
            <CardContent className="pt-6">
              <div className="text-3xl font-bold text-white">{servers.length}</div>
              <div className="text-slate-400 text-sm mt-1">服务器</div>
            </CardContent>
          </Card>
          <Card className="bg-slate-800 border-slate-700">
            <CardContent className="pt-6">
              <div className="text-3xl font-bold text-white">{deployments.length}</div>
              <div className="text-slate-400 text-sm mt-1">部署记录</div>
            </CardContent>
          </Card>
          <Card className="bg-slate-800 border-slate-700">
            <CardContent className="pt-6">
              <div className="text-3xl font-bold text-white">
                {deployments.filter(d => d.status === 'completed').length}
              </div>
              <div className="text-slate-400 text-sm mt-1">成功部署</div>
            </CardContent>
          </Card>
          <Card className="bg-slate-800 border-slate-700">
            <CardContent className="pt-6">
              <div className="text-3xl font-bold text-white">
                {user?.max_servers || 3}
              </div>
              <div className="text-slate-400 text-sm mt-1">服务器配额</div>
            </CardContent>
          </Card>
        </div>

        {/* Quick Actions */}
        <div className="grid md:grid-cols-2 gap-6 mb-8">
          {/* Servers */}
          <Card className="bg-slate-800 border-slate-700">
            <CardHeader>
              <CardTitle className="text-white">服务器</CardTitle>
              <CardDescription className="text-slate-400">
                管理您托管的服务器
              </CardDescription>
            </CardHeader>
            <CardContent>
              {servers.length === 0 ? (
                <div className="text-slate-400 text-sm mb-4">
                  暂无服务器，添加您的第一台服务器
                </div>
              ) : (
                <div className="space-y-2">
                  {servers.slice(0, 3).map((server) => (
                    <div
                      key={server.id}
                      className="flex items-center justify-between p-3 bg-slate-700 rounded-lg"
                    >
                      <div>
                        <div className="text-white font-medium">{server.name}</div>
                        <div className="text-slate-400 text-sm">
                          {server.host}:{server.port}
                        </div>
                      </div>
                      <span
                        className={`px-2 py-1 rounded text-xs ${
                          server.status === 'online'
                            ? 'bg-green-500/20 text-green-400'
                            : 'bg-slate-600 text-slate-400'
                        }`}
                      >
                        {server.status}
                      </span>
                    </div>
                  ))}
                </div>
              )}
              <Link
                href="/dashboard/servers"
                className="text-blue-400 hover:text-blue-300 text-sm mt-4 inline-block"
              >
                查看全部 →
              </Link>
            </CardContent>
          </Card>

          {/* Deployments */}
          <Card className="bg-slate-800 border-slate-700">
            <CardHeader>
              <CardTitle className="text-white">部署</CardTitle>
              <CardDescription className="text-slate-400">
                创建和管理部署任务
              </CardDescription>
            </CardHeader>
            <CardContent>
              {deployments.length === 0 ? (
                <div className="text-slate-400 text-sm mb-4">
                  暂无部署记录，创建您的第一个部署
                </div>
              ) : (
                <div className="space-y-2">
                  {deployments.slice(0, 3).map((deployment) => (
                    <div
                      key={deployment.id}
                      className="flex items-center justify-between p-3 bg-slate-700 rounded-lg"
                    >
                      <div className="flex-1">
                        <div className="text-white font-medium truncate">
                          {deployment.project_name}
                        </div>
                        <div className="text-slate-400 text-sm">
                          {new Date(deployment.created_at).toLocaleDateString()}
                        </div>
                      </div>
                      <span
                        className={`px-2 py-1 rounded text-xs ${
                          deployment.status === 'completed'
                            ? 'bg-green-500/20 text-green-400'
                            : deployment.status === 'failed'
                            ? 'bg-red-500/20 text-red-400'
                            : 'bg-yellow-500/20 text-yellow-400'
                        }`}
                      >
                        {deployment.status}
                      </span>
                    </div>
                  ))}
                </div>
              )}
              <Link
                href="/dashboard/deployments"
                className="text-blue-400 hover:text-blue-300 text-sm mt-4 inline-block"
              >
                查看全部 →
              </Link>
            </CardContent>
          </Card>
        </div>

        {/* New Deployment */}
        <Card className="bg-slate-800 border-slate-700">
          <CardHeader>
            <CardTitle className="text-white">快速部署</CardTitle>
            <CardDescription className="text-slate-400">
              输入 GitHub 项目 URL，AI 将自动分析并部署
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Link
              href="/dashboard/deployments/new"
              className="inline-flex items-center px-6 py-3 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition"
            >
              <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              创建新部署
            </Link>
          </CardContent>
        </Card>
      </main>
    </div>
  )
}

// Simple Card components for this page
function Card({ className, children }: any) {
  return <div className={className}>{children}</div>
}

function CardHeader({ children }: any) {
  return <div className="p-6 pb-3">{children}</div>
}

function CardTitle({ children, className }: any) {
  return <h3 className={`text-xl font-semibold ${className}`}>{children}</h3>
}

function CardDescription({ children, className }: any) {
  return <p className={`text-sm ${className}`}>{children}</p>
}

function CardContent({ children, className }: any) {
  return <div className={`p-6 pt-3 ${className}`}>{children}</div>
}

export default function Home() {
  return (
    <main className="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800">
      {/* Hero Section */}
      <div className="container mx-auto px-4 py-16">
        <div className="text-center">
          <h1 className="text-5xl font-bold text-white mb-6">
            ServerMind
          </h1>
          <p className="text-xl text-slate-300 mb-8">
            AI 驱动的自动化服务器部署平台
          </p>
          <p className="text-lg text-slate-400 mb-12 max-w-2xl mx-auto">
            输入 GitHub 项目 URL，AI 自动分析并部署到您的服务器。
            支持多 Agent 协作、智能故障诊断、知识库自动进化。
          </p>

          <div className="flex gap-4 justify-center">
            <a
              href="/register"
              className="px-8 py-3 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition"
            >
              免费开始
            </a>
            <a
              href="/login"
              className="px-8 py-3 bg-slate-700 text-white rounded-lg font-medium hover:bg-slate-600 transition"
            >
              登录
            </a>
          </div>
        </div>

        {/* Features */}
        <div className="grid md:grid-cols-3 gap-8 mt-24">
          <div className="p-6 bg-slate-800 rounded-xl border border-slate-700">
            <div className="w-12 h-12 bg-blue-600 rounded-lg flex items-center justify-center mb-4">
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-white mb-2">一键部署</h3>
            <p className="text-slate-400">
              只需输入 GitHub 项目 URL，AI 自动分析项目结构、生成部署配置、执行部署流程
            </p>
          </div>

          <div className="p-6 bg-slate-800 rounded-xl border border-slate-700">
            <div className="w-12 h-12 bg-green-600 rounded-lg flex items-center justify-center mb-4">
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-white mb-2">服务器托管</h3>
            <p className="text-slate-400">
              安全托管您的服务器，支持 Ubuntu、CentOS 等主流 Linux 发行版
            </p>
          </div>

          <div className="p-6 bg-slate-800 rounded-xl border border-slate-700">
            <div className="w-12 h-12 bg-purple-600 rounded-lg flex items-center justify-center mb-4">
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-white mb-2">AI 智能诊断</h3>
            <p className="text-slate-400">
              多轮对话排错，自动分析错误日志，从知识库检索相似案例，智能修复问题
            </p>
          </div>
        </div>

        {/* How it works */}
        <div className="mt-24">
          <h2 className="text-3xl font-bold text-white text-center mb-12">工作流程</h2>
          <div className="grid md:grid-cols-4 gap-6">
            <div className="text-center">
              <div className="w-16 h-16 bg-blue-600 rounded-full flex items-center justify-center mx-auto mb-4 text-2xl font-bold text-white">
                1
              </div>
              <h4 className="text-lg font-semibold text-white mb-2">注册账号</h4>
              <p className="text-slate-400 text-sm">创建免费账户，获得 3 台服务器额度</p>
            </div>
            <div className="text-center">
              <div className="w-16 h-16 bg-blue-600 rounded-full flex items-center justify-center mx-auto mb-4 text-2xl font-bold text-white">
                2
              </div>
              <h4 className="text-lg font-semibold text-white mb-2">托管服务器</h4>
              <p className="text-slate-400 text-sm">输入服务器 IP 和密码，安全托管</p>
            </div>
            <div className="text-center">
              <div className="w-16 h-16 bg-blue-600 rounded-full flex items-center justify-center mx-auto mb-4 text-2xl font-bold text-white">
                3
              </div>
              <h4 className="text-lg font-semibold text-white mb-2">输入项目</h4>
              <p className="text-slate-400 text-sm">粘贴 GitHub 项目 URL，AI 自动分析</p>
            </div>
            <div className="text-center">
              <div className="w-16 h-16 bg-blue-600 rounded-full flex items-center justify-center mx-auto mb-4 text-2xl font-bold text-white">
                4
              </div>
              <h4 className="text-lg font-semibold text-white mb-2">自动部署</h4>
              <p className="text-slate-400 text-sm">AI 执行部署，实时查看进度和日志</p>
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <footer className="border-t border-slate-700 mt-24 py-8">
        <div className="container mx-auto px-4 text-center text-slate-400">
          <p>&copy; 2026 ServerMind. MIT License.</p>
        </div>
      </footer>
    </main>
  )
}

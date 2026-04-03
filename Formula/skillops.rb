class Skillops < Formula
  desc "Lightweight CLI to manage AI agent skills using symlinks"
  homepage "https://github.com/leodinhsa/skillops"
  url "https://github.com/leodinhsa/skillops/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "REPLACE_WITH_ACTUAL_SHA256" # Temporary placeholder
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-ldflags", "-X skillops/internal/config.Version=v#{version}", "-o", bin/"skillops", "main.go"
  end

  test do
    system "#{bin}/skillops", "--help"
  end

  def caveats
    <<~EOS
      🚀 skillops installed successfully!

      Quick start:
        skillops pull <repo-url>   — download a skill repo
        skillops init              — declare which IDEs this project uses
        skillops add               — link skills into your IDEs
        skillops status            — see what's linked

      Upgrading from v1? Run in each project:
        skillops init
        skillops sync

      Your skill data lives in ~/.skillops/ and is never removed on uninstall.
      To fully clean up: rm -rf ~/.skillops/
    EOS
  end
end

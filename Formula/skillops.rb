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
      To get started, try running:
        skillops --help
      Or jump right into managing your agentic IDEs:
        skillops agentic
    EOS
  end
end

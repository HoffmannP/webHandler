document.addEventListener('DOMContentLoaded', function main () {
  document.querySelectorAll('button').forEach(b => (b.onclick = function () {
    window.fetch('/' + b.dataset.name)
      .then(r => r.text())
      .then(t => console.log(t))
  }))
})

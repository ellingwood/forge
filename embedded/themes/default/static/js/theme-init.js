(function(){
  var s=localStorage.getItem("theme"),p=window.matchMedia("(prefers-color-scheme:dark)").matches;
  if(s==="dark"||(!s&&p))document.documentElement.classList.add("dark");
})();

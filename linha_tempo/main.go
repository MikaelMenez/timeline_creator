package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// LinhaTempo representa a estrutura raiz do JSON
type LinhaTempo struct {
	Titulo         string          `json:"titulo"`
	Acontecimentos []Acontecimento `json:"acontecimentos"`
}

// Acontecimento representa cada card da linha do tempo
type Acontecimento struct {
	Momento   string `json:"acontecimento"`
	Ordem     uint   `json:"ordem"`
	Data      string `json:"data"`
	Descricao string `json:"descricao"`
}

type RespostaAPI struct {
	Status   string   `json:"status"`
	Arquivos []string `json:"arquivos"`
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Titulo}}</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2 family=Cinzel:wght@500;700&family=Playfair+Display:ital,wght=0,400;0,700;1,400&family=Lora:wght@400;500&display=swap" rel="stylesheet">
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    fontFamily: {
                        header: ['Cinzel', 'serif'],
                        editorial: ['Playfair Display', 'serif'],
                        body: ['Lora', 'serif'],
                    }
                }
            }
        }
    </script>
</head>
<body class="bg-[#f4efe2] text-[#2c221e] min-h-screen py-16 px-6 font-body selection:bg-[#e2d7be]">
    <div class="max-w-3xl mx-auto">

        <header class="text-center mb-16 pb-4">
            <h1 class="text-4xl md:text-5xl font-header font-bold text-[#1f1815] tracking-wide">{{.Titulo}}</h1>
            <div class="w-16 h-[2px] bg-[#8c6d3e] mx-auto mt-6"></div>
        </header>

        <div class="relative border-l-2 border-[#c4b28a] ml-4 md:ml-48 space-y-12">
            {{range .Acontecimentos}}
            <div class="relative pl-8 group">

                <div class="absolute -left-[9px] top-2 bg-[#f4efe2] w-4 h-4 rotate-45 border-2 border-[#8c6d3e] group-hover:bg-[#8c6d3e] transition-colors duration-300 z-10"></div>

                <div class="static md:absolute md:-left-48 md:top-0 md:w-40 md:text-right font-editorial text-xl font-bold text-[#8c6d3e] mb-2 md:mb-0 block leading-none">
                    {{.Data}}
                </div>

                <div class="border-b border-[#e6dcbf] pb-6">
                    <h3 class="text-2xl font-editorial font-bold text-[#1f1815] mb-3 leading-tight tracking-wide">
                        {{.Momento}}
                    </h3>
                    <p class="text-[#42342e] text-base leading-relaxed text-justify font-body">
                        {{.Descricao}}
                    </p>
                </div>
            </div>
            {{end}}
        </div>

    </div>
</body>
</html>`

// Carrega o JSON do disco e devolve o Template pronto e a Struct populada
func le_json_sai_html(path string) (*template.Template, LinhaTempo, error) {
	var lt LinhaTempo

	jason, err := os.ReadFile(path)
	if err != nil {
		return nil, lt, errors.New("erro ao ler arquivo: " + path)
	}

	err = json.Unmarshal(jason, &lt)
	if err != nil {
		return nil, lt, errors.New("erro ao fazer parse do json")
	}

	tmpl, err := template.New("timeline").Parse(htmlTemplate)
	if err != nil {
		return nil, lt, errors.New("erro ao compilar o template html")
	}

	return tmpl, lt, nil
}

// Varre a pasta de JSONs e devolve uma lista limpa com os nomes dos arquivos existentes
func obterListaArquivos() []string {
	var lista []string
	arq, err := os.ReadDir("./linhas_json")
	if err != nil {
		return lista
	}
	for _, arquivo := range arq {
		if !arquivo.IsDir() && filepath.Ext(arquivo.Name()) == ".json" {
			nome := strings.TrimSuffix(arquivo.Name(), ".json")
			if nome != "template" {
				lista = append(lista, nome)
			}
		}
	}
	return lista
}

func main() {
	// Garante que os diretórios necessários existem no Kubuntu
	_ = os.MkdirAll("./linhas_json", 0755)
	_ = os.MkdirAll("./linhas_html", 0755)

	// 1. COMPILAÇÃO INICIAL (Roda ao iniciar o servidor baseando-se nos JSONs existentes)
	nomesArquivos := obterListaArquivos()
	for _, nome := range nomesArquivos {
		caminhoJSON := filepath.Join("linhas_json", nome+".json")
		caminhoHTML := filepath.Join("linhas_html", nome+".html")

		tmpl, dadosLinhaTempo, err := le_json_sai_html(caminhoJSON)
		if err != nil {
			fmt.Println("Erro na compilação inicial:", err)
			continue
		}

		arquivoHTML, err := os.Create(caminhoHTML)
		if err == nil {
			_ = tmpl.Execute(arquivoHTML, dadosLinhaTempo)
			arquivoHTML.Close()
		}
	}
	fmt.Println("✨ Compilação inicial das linhas do tempo concluída!")

	// 2. ROTA PRINCIPAL: Serve a interface insere.html com a lista de arquivos
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			// Permite acessar os arquivos estáticos (ex: /linhas_html/mikael.html)
			http.FileServer(http.Dir(".")).ServeHTTP(w, r)
			return
		}

		tmplInterface, err := template.ParseFiles("insere.html")
		if err != nil {
			http.Error(w, "Erro ao carregar insere.html: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Injeta a lista atualizada de arquivos na interface
		tmplInterface.Execute(w, obterListaArquivos())
	})

	// 3. ROTA DA API: Recebe dados do formulário e cria arquivos em tempo de execução
	http.HandleFunc("/api/gerar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}

		var dados LinhaTempo
		if err := json.NewDecoder(r.Body).Decode(&dados); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}

		nomeBase := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(dados.Titulo)), " ", "_")
		if nomeBase == "" {
			http.Error(w, "Título inválido", http.StatusBadRequest)
			return
		}

		// Salva o arquivo JSON na pasta correta
		caminhoJSON := filepath.Join("linhas_json", nomeBase+".json")
		dadosJson, _ := json.MarshalIndent(dados, "", "  ")
		_ = os.WriteFile(caminhoJSON, dadosJson, 0644)

		// Compila e gera o arquivo HTML correspondente na pasta correta
		caminhoHTML := filepath.Join("linhas_html", nomeBase+".html")
		arquivoHTML, err := os.Create(caminhoHTML)
		if err != nil {
			http.Error(w, "Erro ao criar arquivo HTML", http.StatusInternalServerError)
			return
		}
		defer arquivoHTML.Close()

		tmpl, err := template.New("timeline").Parse(htmlTemplate)
		if err != nil || tmpl.Execute(arquivoHTML, dados) != nil {
			http.Error(w, "Erro ao renderizar template", http.StatusInternalServerError)
			return
		}

		// Devolve a lista atualizada para o Frontend reordenar a tela
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RespostaAPI{
			Status:   "Sucesso",
			Arquivos: obterListaArquivos(),
		})
	})

	fmt.Println("🚀 Servidor rodando em http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Erro ao iniciar o servidor:", err)
	}
}

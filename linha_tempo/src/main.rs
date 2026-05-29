use axum::{
    Router,
    extract::Json,
    response::Html,
    routing::{get, post},
};
use serde::{Deserialize, Serialize};
use std::fs;
use tower_http::services::ServeDir;

#[derive(Serialize, Deserialize, Clone)]
struct Acontecimento {
    acontecimento: String,
    ordem: u32,
    data: String,
    descricao: String,
}

#[derive(Serialize, Deserialize, Clone)]
struct LinhaTempo {
    titulo: String,
    acontecimentos: Vec<Acontecimento>,
}

#[derive(Serialize)]
struct RespostaAPI {
    status: String,
    arquivos: Vec<String>,
}

fn obter_lista_html() -> Vec<String> {
    fs::read_dir("linhas_html")
        .map(|entries| {
            entries
                .filter_map(|e| e.ok())
                .filter(|e| e.path().extension().and_then(|s| s.to_str()) == Some("html"))
                .map(|e| e.path().file_stem().unwrap().to_string_lossy().into_owned())
                .collect()
        })
        .unwrap_or_default()
}

async fn api_gerar(Json(dados): Json<LinhaTempo>) -> Json<RespostaAPI> {
    let nome_base = dados.titulo.to_lowercase().replace(" ", "_");

    // Caminhos compatíveis (PathBuf lida com / vs \)
    let base_path = std::env::current_dir().unwrap();
    let json_path = base_path
        .join("linhas_json")
        .join(format!("{}.json", nome_base));
    let html_path = base_path
        .join("linhas_html")
        .join(format!("{}.html", nome_base));
    let template_path = base_path.join("template.html");

    // Salvar JSON
    let _ = fs::write(json_path, serde_json::to_string_pretty(&dados).unwrap());

    // Salvar HTML
    let template = fs::read_to_string(template_path).expect("template.html não encontrado!");
    let mut blocos_html = String::new();
    for item in dados.acontecimentos {
        blocos_html.push_str(&format!(
            "<div class='evento'><div class='data'>{}</div><div><h2>{}</h2><p>{}</p></div></div>",
            item.data, item.acontecimento, item.descricao
        ));
    }

    let html_final = template
        .replace("{{Titulo}}", &dados.titulo)
        .replace("{{Acontecimentos}}", &blocos_html);
    let _ = fs::write(html_path, html_final);

    Json(RespostaAPI {
        status: "Sucesso".into(),
        arquivos: obter_lista_html(),
    })
}

#[tokio::main]
async fn main() {
    let _ = fs::create_dir_all("linhas_json");
    let _ = fs::create_dir_all("linhas_html");

    let app = Router::new()
        .route(
            "/",
            get(|| async { Html(fs::read_to_string("insere.html").unwrap_or_default()) }),
        )
        .route("/api/arquivos", get(|| async { Json(obter_lista_html()) }))
        .route("/api/gerar", post(api_gerar))
        .nest_service("/linhas_html", ServeDir::new("linhas_html"));

    println!("🚀 Servidor rodando em http://localhost:8080");
    let _ = opener::open("http://localhost:8080");

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080").await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
